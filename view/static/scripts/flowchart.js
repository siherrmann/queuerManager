// Global function to add task card from button click
function addTaskCardFromData(taskJson) {
  const task = JSON.parse(taskJson);
  if (window.flowsAddTaskCard) {
    window.flowsAddTaskCard(task);
  }
}

// Global function to save flow data
async function saveFlowData() {
  const flowRid = document.getElementById("flow-rid")?.value;
  if (!flowRid) {
    alert("No flow to save. Please create a flow first.");
    return;
  }

  const statusEl = document.getElementById("save-status");
  const saveBtn = document.getElementById("save-flow-btn");

  if (statusEl) statusEl.textContent = "Saving...";
  if (saveBtn) saveBtn.disabled = true;

  const flowData = window.getFlowData
    ? window.getFlowData()
    : { parts: [], connections: [] };

  try {
    const response = await fetch(`/api/flow/${flowRid}/data`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        "HX-Request": "true",
      },
      body: JSON.stringify(flowData),
    });

    if (response.ok) {
      const result = await response.json();
      if (statusEl) {
        statusEl.textContent = "Saved!";
        statusEl.className = "text-sm text-green-600";
        setTimeout(() => {
          statusEl.textContent = "";
          statusEl.className = "text-sm text-gray-500";
        }, 2000);
      }
    } else {
      const error = await response.json();
      throw new Error(error.error || "Failed to save");
    }
  } catch (err) {
    console.error("Save error:", err);
    if (statusEl) {
      statusEl.textContent = "Error saving!";
      statusEl.className = "text-sm text-red-600";
    }
  } finally {
    if (saveBtn) saveBtn.disabled = false;
  }
}

(function () {
  // State
  let cards = [];
  let connections = [];
  let cardIdCounter = 0;
  let isPanning = false;
  let isConnecting = false;
  let isDraggingCard = false;
  let panStart = { x: 0, y: 0 };
  let offset = { x: 0, y: 0 };
  let scale = 1;
  let draggedCard = null;
  let dragOffset = { x: 0, y: 0 };
  let connectionStart = null;
  let tempLine = null;

  // DOM Elements
  const whiteboard = document.getElementById("whiteboard");
  const cardsContainer = document.getElementById("cards-container");
  const connectionsLayer = document.getElementById("connections-layer");
  const zoomInBtn = document.getElementById("zoom-in-btn");
  const zoomOutBtn = document.getElementById("zoom-out-btn");
  const zoomResetBtn = document.getElementById("zoom-reset-btn");

  // Expose addTaskCard globally for onclick handlers
  window.flowsAddTaskCard = async function (task) {
    console.log("flowsAddTaskCard called with:", task);
    const centerX = -offset.x / scale + 100;
    const centerY = -offset.y / scale + 100;
    await addTaskCard(
      task,
      centerX + (Math.random() - 0.5) * 100,
      centerY + (Math.random() - 0.5) * 100,
    );
  };

  // Expose addBlockCard globally for onclick handlers
  window.addBlockCard = async function (blockType) {
    console.log("addBlockCard called with:", blockType);
    const centerX = -offset.x / scale + 100;
    const centerY = -offset.y / scale + 100;
    await addBlock(
      blockType,
      centerX + (Math.random() - 0.5) * 100,
      centerY + (Math.random() - 0.5) * 100,
    );
  };

  // Expose getFlowData globally for save function
  window.getFlowData = function () {
    const parts = cards.map((card) => {
      const partData = {
        id: card.id,
        type: card.partType || "task",
        position: { x: card.x, y: card.y },
        config: card.config || {},
      };

      // Add task-specific fields
      if (card.partType === "task" || !card.partType) {
        partData.task_rid = card.taskRid;
        partData.task_key = card.taskKey;
        partData.task_name = card.taskName;
        // Extract timeout_minutes from config for task cards
        if (card.config && card.config.timeout_minutes) {
          const timeout = parseInt(card.config.timeout_minutes, 10);
          if (!isNaN(timeout) && timeout > 0) {
            partData.timeout_minutes = timeout;
          }
        }
      }

      return partData;
    });

    const conns = connections.map((conn) => ({
      from_part_id: conn.fromCard,
      from_param_key: conn.fromParam,
      to_part_id: conn.toCard,
      to_param_key: conn.toParam,
    }));

    return { parts, connections: conns };
  };

  // Initialize
  function init() {
    centerView();
    setupEventListeners();
    loadExistingFlowData();
  }

  // Load existing flow data from hidden input
  async function loadExistingFlowData() {
    const flowDataEl = document.getElementById("flow-data");
    if (!flowDataEl || !flowDataEl.value) return;

    try {
      const flowData = JSON.parse(flowDataEl.value);
      if (!flowData.parts || flowData.parts.length === 0) return;

      // Load parts (cards) - fetch each one from the backend
      const loadPromises = [];
      for (const part of flowData.parts) {
        if (part.type === "task") {
          // Create a task object with the saved data
          const task = {
            rid: part.task_rid,
            key: part.task_key,
            name: part.task_name,
          };

          // Use stored position - await each card fetch
          loadPromises.push(
            addTaskCardFromSavedData(
              part.id,
              task,
              part.position.x,
              part.position.y,
              part.config,
            ),
          );
        } else if (
          part.type === "flow_parameter" ||
          part.type === "const_parameter" ||
          part.type === "if_else" ||
          part.type === "custom_data"
        ) {
          // Load block cards
          loadPromises.push(
            addBlockFromSavedData(
              part.id,
              part.type,
              part.position.x,
              part.position.y,
              part.config,
            ),
          );
        }
      }

      // Wait for all cards to be loaded
      await Promise.all(loadPromises);

      // Load connections after cards are created
      flowData.connections.forEach((conn) => {
        addConnectionFromSavedData(
          conn.from_part_id,
          conn.from_param_key,
          conn.to_part_id,
          conn.to_param_key,
        );
      });

      // Center view on loaded cards
      if (cards.length > 0) {
        centerCardsInView();
      }
    } catch (err) {
      console.error("Error loading flow data:", err);
    }
  }

  function centerView() {
    const rect = whiteboard.getBoundingClientRect();
    offset.x = rect.width / 2;
    offset.y = rect.height / 2;
    updateTransform();
  }

  function updateTransform() {
    cardsContainer.style.transform = `translate(${offset.x}px, ${offset.y}px) scale(${scale})`;
    updateAllConnections();
  }

  function setupEventListeners() {
    // Zoom controls
    zoomInBtn.addEventListener("click", () => zoom(0.2));
    zoomOutBtn.addEventListener("click", () => zoom(-0.2));
    zoomResetBtn.addEventListener("click", () => {
      scale = 1;
      centerView();
      centerCardsInView();
    });

    // Whiteboard panning
    whiteboard.addEventListener("mousedown", onWhiteboardMouseDown);
    whiteboard.addEventListener("mousemove", onWhiteboardMouseMove);
    whiteboard.addEventListener("mouseup", onWhiteboardMouseUp);
    whiteboard.addEventListener("mouseleave", onWhiteboardMouseUp);

    // Mouse wheel zoom
    whiteboard.addEventListener("wheel", onWheel, { passive: false });

    // Touch support
    whiteboard.addEventListener("touchstart", onTouchStart, { passive: false });
    whiteboard.addEventListener("touchmove", onTouchMove, { passive: false });
    whiteboard.addEventListener("touchend", onTouchEnd);
  }

  function zoom(delta) {
    const newScale = Math.max(0.25, Math.min(2, scale + delta));
    scale = newScale;
    updateTransform();
    updateArrowheadSize();
    updateAllConnectionLines();
  }

  function updateArrowheadSize() {
    const marker = document.getElementById("arrowhead");
    if (marker) {
      const baseWidth = 10;
      const baseHeight = 7;
      const scaledWidth = baseWidth * scale;
      const scaledHeight = baseHeight * scale;
      marker.setAttribute("markerWidth", scaledWidth.toString());
      marker.setAttribute("markerHeight", scaledHeight.toString());
      // viewBox stays the same, polygon scales via markerWidth/markerHeight
    }
  }

  function updateAllConnectionLines() {
    connections.forEach((conn) => {
      if (conn.line) {
        const fromPos = getOutputPosition(conn.fromCard, conn.fromOutput);
        const toPos = getInputPosition(conn.toCard, conn.toInput);
        if (fromPos && toPos) {
          updateConnectionLine(
            conn.line,
            fromPos.x,
            fromPos.y,
            toPos.x,
            toPos.y,
            false,
          );
        }
      }
    });
  }

  function onWheel(e) {
    e.preventDefault();
    const delta = e.deltaY > 0 ? -0.1 : 0.1;
    zoom(delta);
  }

  function onWhiteboardMouseDown(e) {
    if (
      e.target.closest(".flow-card") ||
      e.target.closest(".connection-point")
    ) {
      return;
    }

    isPanning = true;
    panStart.x = e.clientX - offset.x;
    panStart.y = e.clientY - offset.y;
    whiteboard.style.cursor = "grabbing";
  }

  function onWhiteboardMouseMove(e) {
    if (isPanning) {
      offset.x = e.clientX - panStart.x;
      offset.y = e.clientY - panStart.y;
      updateTransform();
    }

    if (isDraggingCard && draggedCard) {
      const rect = whiteboard.getBoundingClientRect();
      const x = (e.clientX - rect.left - offset.x) / scale - dragOffset.x;
      const y = (e.clientY - rect.top - offset.y) / scale - dragOffset.y;

      const card = cards.find((c) => c.id === draggedCard.dataset.cardId);
      if (card) {
        card.x = x;
        card.y = y;
        draggedCard.style.left = `${x}px`;
        draggedCard.style.top = `${y}px`;
        updateAllConnections();
      }
    }
  }

  // Document-level handlers for connection drawing (so we can drag over cards)
  function onDocumentMouseMove(e) {
    if (!isConnecting || !connectionStart || !tempLine) return;

    const rect = whiteboard.getBoundingClientRect();
    const mouseX = (e.clientX - rect.left - offset.x) / scale;
    const mouseY = (e.clientY - rect.top - offset.y) / scale;

    const startCard = cards.find((c) => c.id === connectionStart.cardId);
    if (startCard) {
      const startCardEl = document.querySelector(
        `[data-card-id="${connectionStart.cardId}"]`,
      );
      const outputPoint = startCardEl?.querySelector(
        `[data-param-key="${connectionStart.paramKey}"].connection-output`,
      );
      if (outputPoint && startCardEl) {
        const cardRect = startCardEl.getBoundingClientRect();
        const pointRect = outputPoint.getBoundingClientRect();
        const startX =
          startCard.x +
          (pointRect.left - cardRect.left + pointRect.width) / scale +
          12;
        const startY =
          startCard.y +
          (pointRect.top - cardRect.top + pointRect.height / 2) / scale;
        updateConnectionLine(tempLine, startX, startY, mouseX, mouseY, true);
      }
    }
  }

  function onDocumentMouseUp(e) {
    if (isConnecting) {
      // Check if we're over an input connection point
      const target = document.elementFromPoint(e.clientX, e.clientY);
      const inputPoint = target?.closest(".connection-input");
      if (inputPoint && connectionStart) {
        const targetCard = inputPoint.closest(".flow-card");
        const targetParamKey = inputPoint.dataset.paramKey;
        if (
          targetCard &&
          targetCard.dataset.cardId !== connectionStart.cardId
        ) {
          createConnection(
            connectionStart.cardId,
            connectionStart.paramKey,
            targetCard.dataset.cardId,
            targetParamKey,
          );
        }
      }

      // Clean up temp line and state
      if (tempLine) {
        tempLine.remove();
        tempLine = null;
      }
      isConnecting = false;
      connectionStart = null;

      // Remove document-level listeners
      document.removeEventListener("mousemove", onDocumentMouseMove);
      document.removeEventListener("mouseup", onDocumentMouseUp);
    }
  }

  function onWhiteboardMouseUp(e) {
    if (isPanning) {
      isPanning = false;
      whiteboard.style.cursor = "grab";
    }

    if (isDraggingCard) {
      isDraggingCard = false;
      draggedCard = null;
    }
  }

  // Touch handlers
  let lastTouchDist = 0;

  function onTouchStart(e) {
    if (e.touches.length === 1) {
      const touch = e.touches[0];
      if (
        !e.target.closest(".flow-card") &&
        !e.target.closest(".connection-point")
      ) {
        isPanning = true;
        panStart.x = touch.clientX - offset.x;
        panStart.y = touch.clientY - offset.y;
      }
    } else if (e.touches.length === 2) {
      isPanning = false;
      lastTouchDist = Math.hypot(
        e.touches[0].clientX - e.touches[1].clientX,
        e.touches[0].clientY - e.touches[1].clientY,
      );
    }
  }

  function onTouchMove(e) {
    e.preventDefault();

    if (e.touches.length === 1 && isPanning) {
      const touch = e.touches[0];
      offset.x = touch.clientX - panStart.x;
      offset.y = touch.clientY - panStart.y;
      updateTransform();
    } else if (e.touches.length === 2) {
      const dist = Math.hypot(
        e.touches[0].clientX - e.touches[1].clientX,
        e.touches[0].clientY - e.touches[1].clientY,
      );
      const delta = dist - lastTouchDist;
      zoom(delta);
      lastTouchDist = dist;
    }
  }

  function onTouchEnd() {
    isPanning = false;
  }

  // Fetch a flow part card from the backend and add it to the whiteboard
  async function addTaskCard(task, x, y) {
    const id = `card-${cardIdCounter++}`;
    console.log("addTaskCard called with:", { task, x, y, id });

    try {
      const params = new URLSearchParams({
        type: "task",
        task_rid: task.rid,
        id: id,
        x: x.toString(),
        y: y.toString(),
      });

      console.log("Fetching from:", `/flow/part?${params}`);
      const response = await fetch(`/flow/part?${params}`);
      console.log("Response status:", response.status, response.ok);
      if (!response.ok) {
        console.error("Failed to fetch flow part:", response.statusText);
        return null;
      }

      const html = await response.text();
      console.log("Received HTML length:", html.length);
      const template = document.createElement("template");
      template.innerHTML = html.trim();
      const cardEl = template.content.firstChild;
      console.log("Card element:", cardEl);

      // Set position
      cardEl.style.left = `${x}px`;
      cardEl.style.top = `${y}px`;

      // Extract inputs and outputs from the card element
      const inputs = [];
      const outputs = [];
      cardEl.querySelectorAll(".connection-input").forEach((el) => {
        inputs.push(el.dataset.paramKey);
      });
      cardEl.querySelectorAll(".connection-output").forEach((el) => {
        outputs.push(el.dataset.paramKey);
      });

      // Store card data
      const cardData = {
        id,
        x,
        y,
        partType: "task",
        taskKey: cardEl.dataset.taskKey || task.key,
        taskName: cardEl.dataset.taskName || task.name,
        taskRid: cardEl.dataset.taskRid || task.rid,
        inputs,
        outputs,
        config: {},
      };
      cards.push(cardData);

      // Attach event listeners
      attachCardEventListeners(cardEl, id);

      cardsContainer.appendChild(cardEl);
      return cardEl;
    } catch (err) {
      console.error("Error fetching flow part:", err);
      return null;
    }
  }

  function deleteCard(cardId) {
    connections = connections.filter((conn) => {
      if (conn.fromCard === cardId || conn.toCard === cardId) {
        const line = document.querySelector(
          `[data-connection-id="${conn.id}"]`,
        );
        if (line) line.remove();
        return false;
      }
      return true;
    });

    cards = cards.filter((c) => c.id !== cardId);

    const cardEl = document.querySelector(`[data-card-id="${cardId}"]`);
    if (cardEl) cardEl.remove();
  }

  // Add a task card from saved flow data (with specific ID)
  async function addTaskCardFromSavedData(savedId, task, x, y, config) {
    // Parse the ID to update counter if needed
    const idNum = parseInt(savedId.replace("card-", ""), 10);
    if (!isNaN(idNum) && idNum >= cardIdCounter) {
      cardIdCounter = idNum + 1;
    }

    const id = savedId;

    try {
      const params = new URLSearchParams({
        type: "task",
        id: id,
        x: x.toString(),
        y: y.toString(),
      });

      // Use task_rid if available, otherwise use task_key
      if (task.rid) {
        params.set("task_rid", task.rid);
      } else if (task.key) {
        params.set("task_key", task.key);
      }

      const response = await fetch(`/flow/part?${params}`);
      if (!response.ok) {
        console.error("Failed to fetch flow part:", response.statusText);
        return null;
      }

      const html = await response.text();
      const template = document.createElement("template");
      template.innerHTML = html.trim();
      const cardEl = template.content.firstChild;

      // Set position
      cardEl.style.left = `${x}px`;
      cardEl.style.top = `${y}px`;

      // Extract inputs and outputs from the card element
      const inputs = [];
      const outputs = [];
      cardEl.querySelectorAll(".connection-input").forEach((el) => {
        inputs.push(el.dataset.paramKey);
      });
      cardEl.querySelectorAll(".connection-output").forEach((el) => {
        outputs.push(el.dataset.paramKey);
      });

      // Store card data
      const cardData = {
        id,
        x,
        y,
        partType: "task",
        taskKey: cardEl.dataset.taskKey || task.key,
        taskName: cardEl.dataset.taskName || task.name,
        taskRid: cardEl.dataset.taskRid || task.rid,
        inputs,
        outputs,
        config: config || {},
      };
      cards.push(cardData);

      // Attach event listeners
      attachCardEventListeners(cardEl, id);

      cardsContainer.appendChild(cardEl);
      return cardEl;
    } catch (err) {
      console.error("Error fetching flow part:", err);
      return null;
    }
  }

  // Add a new block card (flow_parameter, const_parameter, if_else)
  async function addBlock(blockType, x, y) {
    const id = `card-${cardIdCounter++}`;
    console.log("addBlock called with:", { blockType, x, y, id });

    try {
      const params = new URLSearchParams({
        type: blockType,
        id: id,
        x: x.toString(),
        y: y.toString(),
      });

      const response = await fetch(`/flow/part?${params}`);
      if (!response.ok) {
        console.error("Failed to fetch block part:", response.statusText);
        return null;
      }

      const html = await response.text();
      const template = document.createElement("template");
      template.innerHTML = html.trim();
      const cardEl = template.content.firstChild;

      // Set position
      cardEl.style.left = `${x}px`;
      cardEl.style.top = `${y}px`;

      // Extract inputs and outputs
      const inputs = [];
      const outputs = [];
      cardEl.querySelectorAll(".connection-input").forEach((el) => {
        inputs.push(el.dataset.paramKey);
      });
      cardEl.querySelectorAll(".connection-output").forEach((el) => {
        outputs.push(el.dataset.paramKey);
      });

      // Store card data
      const cardData = {
        id,
        x,
        y,
        partType: blockType,
        inputs,
        outputs,
        config: {},
      };
      cards.push(cardData);

      // Attach event listeners
      attachCardEventListeners(cardEl, id);

      cardsContainer.appendChild(cardEl);
      return cardEl;
    } catch (err) {
      console.error("Error fetching block part:", err);
      return null;
    }
  }

  // Add a block from saved flow data
  async function addBlockFromSavedData(savedId, blockType, x, y, config) {
    const idNum = parseInt(savedId.replace("card-", ""), 10);
    if (!isNaN(idNum) && idNum >= cardIdCounter) {
      cardIdCounter = idNum + 1;
    }

    const id = savedId;

    try {
      const params = new URLSearchParams({
        type: blockType,
        id: id,
        x: x.toString(),
        y: y.toString(),
      });

      // Add config values to params
      if (config) {
        for (const [key, value] of Object.entries(config)) {
          if (value !== undefined && value !== null) {
            params.set(key, String(value));
          }
        }
      }

      const response = await fetch(`/flow/part?${params}`);
      if (!response.ok) {
        console.error("Failed to fetch block part:", response.statusText);
        return null;
      }

      const html = await response.text();
      const template = document.createElement("template");
      template.innerHTML = html.trim();
      const cardEl = template.content.firstChild;

      // Set position
      cardEl.style.left = `${x}px`;
      cardEl.style.top = `${y}px`;

      // Extract inputs and outputs
      const inputs = [];
      const outputs = [];
      cardEl.querySelectorAll(".connection-input").forEach((el) => {
        inputs.push(el.dataset.paramKey);
      });
      cardEl.querySelectorAll(".connection-output").forEach((el) => {
        outputs.push(el.dataset.paramKey);
      });

      // Store card data
      const cardData = {
        id,
        x,
        y,
        partType: blockType,
        inputs,
        outputs,
        config: config || {},
      };
      cards.push(cardData);

      // Attach event listeners
      attachCardEventListeners(cardEl, id);

      cardsContainer.appendChild(cardEl);
      return cardEl;
    } catch (err) {
      console.error("Error fetching block part:", err);
      return null;
    }
  }

  // Attach event listeners to a card element
  function attachCardEventListeners(cardEl, cardId) {
    // Card drag handling
    cardEl.addEventListener("mousedown", (e) => {
      if (
        e.target.closest(".connection-point") ||
        e.target.closest(".delete-card-btn") ||
        e.target.closest(".config-input")
      ) {
        return;
      }
      e.stopPropagation();
      isDraggingCard = true;
      draggedCard = cardEl;

      const rect = whiteboard.getBoundingClientRect();
      const cardX = parseFloat(cardEl.style.left);
      const cardY = parseFloat(cardEl.style.top);
      dragOffset.x = (e.clientX - rect.left - offset.x) / scale - cardX;
      dragOffset.y = (e.clientY - rect.top - offset.y) / scale - cardY;

      cardEl.style.zIndex = cardIdCounter + 100;
    });

    // Config input handlers - update card config when inputs change
    cardEl.querySelectorAll(".config-input").forEach((input) => {
      input.addEventListener("change", () => {
        const card = cards.find((c) => c.id === cardId);
        if (card) {
          const configKey = input.dataset.configKey;
          if (configKey) {
            if (!card.config) card.config = {};
            card.config[configKey] = input.value;
          }
        }
      });
      // Also listen for input events for immediate updates
      input.addEventListener("input", () => {
        const card = cards.find((c) => c.id === cardId);
        if (card) {
          const configKey = input.dataset.configKey;
          if (configKey) {
            if (!card.config) card.config = {};
            card.config[configKey] = input.value;
          }
        }
      });
    });

    // Output connection point handlers
    cardEl.querySelectorAll(".connection-output").forEach((outputPoint) => {
      outputPoint.addEventListener("mousedown", (e) => {
        e.stopPropagation();
        e.preventDefault();
        isConnecting = true;
        connectionStart = {
          cardId: cardId,
          paramKey: outputPoint.dataset.paramKey,
          paramType: outputPoint.dataset.paramType,
        };

        tempLine = document.createElementNS(
          "http://www.w3.org/2000/svg",
          "path",
        );
        tempLine.setAttribute("stroke", "#6b7280");
        tempLine.setAttribute("stroke-width", (2 * scale).toString());
        tempLine.setAttribute("fill", "none");
        tempLine.setAttribute("stroke-dasharray", "5,5");
        tempLine.classList.add("temp-connection");
        connectionsLayer.appendChild(tempLine);

        // Add document-level listeners for connection drawing
        document.addEventListener("mousemove", onDocumentMouseMove);
        document.addEventListener("mouseup", onDocumentMouseUp);
      });
    });

    // Delete button
    const deleteBtn = cardEl.querySelector(".delete-card-btn");
    if (deleteBtn) {
      deleteBtn.addEventListener("click", () => {
        deleteCard(cardId);
      });
    }
  }

  // Add connection from saved flow data
  function addConnectionFromSavedData(
    fromCardId,
    fromParam,
    toCardId,
    toParam,
  ) {
    createConnection(fromCardId, fromParam, toCardId, toParam);
  }

  function createConnection(fromCardId, fromParam, toCardId, toParam) {
    // Check if connection already exists
    if (
      connections.some(
        (c) =>
          c.fromCard === fromCardId &&
          c.fromParam === fromParam &&
          c.toCard === toCardId &&
          c.toParam === toParam,
      )
    ) {
      return;
    }

    const connId = `conn-${fromCardId}-${fromParam}-${toCardId}-${toParam}`;
    connections.push({
      id: connId,
      fromCard: fromCardId,
      fromParam: fromParam,
      toCard: toCardId,
      toParam: toParam,
    });

    const line = document.createElementNS("http://www.w3.org/2000/svg", "path");
    line.setAttribute("stroke", "#6b7280");
    line.setAttribute("stroke-width", (2 * scale).toString());
    line.setAttribute("fill", "none");
    line.setAttribute("marker-end", "url(#arrowhead)");
    line.dataset.connectionId = connId;
    line.classList.add("connection-line");

    line.style.pointerEvents = "stroke";
    line.style.cursor = "pointer";
    line.addEventListener("click", () => {
      if (confirm("Delete this connection?")) {
        connections = connections.filter((c) => c.id !== connId);
        line.remove();
      }
    });

    connectionsLayer.appendChild(line);
    updateAllConnections();
  }

  function updateAllConnections() {
    connections.forEach((conn) => {
      const line = document.querySelector(`[data-connection-id="${conn.id}"]`);
      if (!line) return;

      const fromCard = cards.find((c) => c.id === conn.fromCard);
      const toCard = cards.find((c) => c.id === conn.toCard);

      if (fromCard && toCard) {
        const fromCardEl = document.querySelector(
          `[data-card-id="${conn.fromCard}"]`,
        );
        const toCardEl = document.querySelector(
          `[data-card-id="${conn.toCard}"]`,
        );

        if (fromCardEl && toCardEl) {
          // Find specific parameter connection points
          const outputPoint = fromCardEl.querySelector(
            `[data-param-key="${conn.fromParam}"].connection-output`,
          );
          const inputPoint = toCardEl.querySelector(
            `[data-param-key="${conn.toParam}"].connection-input`,
          );

          if (outputPoint && inputPoint) {
            const fromCardRect = fromCardEl.getBoundingClientRect();
            const toCardRect = toCardEl.getBoundingClientRect();
            const outputRect = outputPoint.getBoundingClientRect();
            const inputRect = inputPoint.getBoundingClientRect();

            // Calculate positions relative to card
            const startX =
              fromCard.x +
              (outputRect.left - fromCardRect.left + outputRect.width) / scale +
              8;
            const startY =
              fromCard.y +
              (outputRect.top - fromCardRect.top + outputRect.height / 2) /
                scale;
            const endX =
              toCard.x + (inputRect.left - toCardRect.left) / scale - 18;
            const endY =
              toCard.y +
              (inputRect.top - toCardRect.top + inputRect.height / 2) / scale;

            updateConnectionLine(line, startX, startY, endX, endY, false);
          }
        }
      }
    });
  }

  function updateConnectionLine(line, x1, y1, x2, y2, isTemp) {
    const screenX1 = x1 * scale + offset.x;
    const screenY1 = y1 * scale + offset.y;
    const screenX2 = x2 * scale + offset.x;
    const screenY2 = y2 * scale + offset.y;

    const dx = screenX2 - screenX1;
    const controlOffset = Math.min(Math.abs(dx) * 0.5, 100);

    const path = `M ${screenX1} ${screenY1} C ${screenX1 + controlOffset} ${screenY1}, ${screenX2 - controlOffset} ${screenY2}, ${screenX2} ${screenY2}`;
    line.setAttribute("d", path);

    // Scale stroke width with zoom
    line.setAttribute("stroke-width", (2 * scale).toString());

    // Update marker size based on scale
    updateArrowheadSize();
  }

  function centerCardsInView() {
    if (cards.length === 0) return;

    let minX = Infinity,
      minY = Infinity,
      maxX = -Infinity,
      maxY = -Infinity;
    cards.forEach((card) => {
      minX = Math.min(minX, card.x);
      minY = Math.min(minY, card.y);
      maxX = Math.max(maxX, card.x + 280);
      maxY = Math.max(maxY, card.y + 150);
    });

    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;

    const rect = whiteboard.getBoundingClientRect();
    offset.x = rect.width / 2 - centerX * scale;
    offset.y = rect.height / 2 - centerY * scale;

    updateTransform();
  }

  // Initialize when DOM is ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
