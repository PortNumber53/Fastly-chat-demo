/**
 * Fastly Chat Demo - Client Application
 * Real-time chat using WebSocket with optional Fastly Fanout delivery
 */

(function () {
    'use strict';

    // ---- State ----
    let ws = null;
    let currentUser = '';
    let currentRoom = 'general';
    let reconnectAttempts = 0;
    const MAX_RECONNECT = 10;
    const RECONNECT_DELAY = 2000;

    // ---- DOM Elements ----
    const loginScreen = document.getElementById('login-screen');
    const chatScreen = document.getElementById('chat-screen');
    const loginForm = document.getElementById('login-form');
    const usernameInput = document.getElementById('username');
    const roomInput = document.getElementById('room');
    const messageForm = document.getElementById('message-form');
    const messageInput = document.getElementById('message-input');
    const messagesDiv = document.getElementById('messages');
    const roomNameEl = document.getElementById('current-room-name');
    const headerRoomName = document.getElementById('header-room-name');
    const userCountEl = document.getElementById('user-count');
    const headerUserCount = document.getElementById('header-user-count');
    const roomListEl = document.getElementById('room-list');
    const connectionDot = document.getElementById('connection-dot');
    const connectionText = document.getElementById('connection-text');
    const leaveBtn = document.getElementById('leave-btn');
    const fanoutBadge = document.getElementById('fanout-badge');
    const sidebarToggle = document.getElementById('sidebar-toggle');
    const sidebar = document.querySelector('.sidebar');

    // ---- Login ----

    loginForm.addEventListener('submit', function (e) {
        e.preventDefault();
        currentUser = usernameInput.value.trim();
        currentRoom = roomInput.value.trim() || 'general';

        if (!currentUser) return;

        localStorage.setItem('chat-username', currentUser);
        showChat();
        connectWebSocket();
    });

    // Restore saved username
    const savedUser = localStorage.getItem('chat-username');
    if (savedUser) {
        usernameInput.value = savedUser;
    }

    // ---- Chat ----

    function showChat() {
        loginScreen.classList.add('hidden');
        chatScreen.classList.remove('hidden');
        updateRoomDisplay();
        messageInput.focus();
        checkHealth();
    }

    function showLogin() {
        chatScreen.classList.add('hidden');
        loginScreen.classList.remove('hidden');
    }

    // ---- WebSocket ----

    function connectWebSocket() {
        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${protocol}//${location.host}/ws?room=${encodeURIComponent(currentRoom)}&username=${encodeURIComponent(currentUser)}`;

        updateConnectionStatus('connecting');
        ws = new WebSocket(wsURL);

        ws.onopen = function () {
            reconnectAttempts = 0;
            updateConnectionStatus('connected');
            addSystemMessage('Connected to chat server');
        };

        ws.onmessage = function (event) {
            try {
                const msg = JSON.parse(event.data);
                handleMessage(msg);
            } catch (err) {
                console.error('Failed to parse message:', err);
            }
        };

        ws.onclose = function () {
            updateConnectionStatus('disconnected');
            addSystemMessage('Disconnected from server');
            attemptReconnect();
        };

        ws.onerror = function () {
            updateConnectionStatus('disconnected');
        };
    }

    function attemptReconnect() {
        if (reconnectAttempts >= MAX_RECONNECT) {
            addSystemMessage('Max reconnection attempts reached. Please refresh.');
            return;
        }
        reconnectAttempts++;
        const delay = RECONNECT_DELAY * Math.min(reconnectAttempts, 5);
        addSystemMessage(`Reconnecting in ${delay / 1000}s... (attempt ${reconnectAttempts})`);
        setTimeout(connectWebSocket, delay);
    }

    function disconnectWebSocket() {
        if (ws) {
            ws.onclose = null; // prevent auto reconnect
            ws.close();
            ws = null;
        }
    }

    // ---- Message Handling ----

    messageForm.addEventListener('submit', function (e) {
        e.preventDefault();
        const text = messageInput.value.trim();
        if (!text || !ws || ws.readyState !== WebSocket.OPEN) return;

        const msg = {
            content: text,
        };

        ws.send(JSON.stringify(msg));
        messageInput.value = '';
        messageInput.focus();
    });

    function handleMessage(msg) {
        switch (msg.type) {
            case 'chat':
                addChatMessage(msg);
                break;
            case 'join':
                addSystemMessage(`${msg.username} joined the chat`);
                break;
            case 'leave':
            case 'system':
                addSystemMessage(msg.content);
                break;
            default:
                addChatMessage(msg);
        }
    }

    function addChatMessage(msg) {
        const isSelf = msg.username === currentUser;
        const el = document.createElement('div');
        el.className = `message${isSelf ? ' self' : ''}`;

        const initials = (msg.username || '?').substring(0, 2).toUpperCase();
        const time = formatTime(msg.time);

        el.innerHTML = `
            <div class="message-avatar">${initials}</div>
            <div class="message-body">
                <div class="message-header">
                    <span class="message-username">${escapeHtml(msg.username)}</span>
                    <span class="message-time">${time}</span>
                </div>
                <div class="message-text">${escapeHtml(msg.content)}</div>
            </div>
        `;

        messagesDiv.appendChild(el);
        scrollToBottom();
    }

    function addSystemMessage(text) {
        const el = document.createElement('div');
        el.className = 'message system';
        el.innerHTML = `
            <div class="message-body">
                <div class="message-text">${escapeHtml(text)}</div>
            </div>
        `;
        messagesDiv.appendChild(el);
        scrollToBottom();
    }

    function scrollToBottom() {
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
    }

    function formatTime(ts) {
        if (!ts) return '';
        const d = new Date(ts);
        return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    // ---- Connection Status ----

    function updateConnectionStatus(status) {
        connectionDot.className = 'dot ' + status;
        switch (status) {
            case 'connected':
                connectionText.textContent = 'Connected';
                break;
            case 'connecting':
                connectionText.textContent = 'Connecting...';
                break;
            case 'disconnected':
                connectionText.textContent = 'Disconnected';
                break;
        }
    }

    // ---- Room Management ----

    function updateRoomDisplay() {
        roomNameEl.textContent = `#${currentRoom}`;
        headerRoomName.textContent = `#${currentRoom}`;
        document.title = `#${currentRoom} - Fastly Chat`;
    }

    leaveBtn.addEventListener('click', function () {
        disconnectWebSocket();
        messagesDiv.innerHTML = '';
        showLogin();
    });

    // ---- Sidebar ----

    sidebarToggle.addEventListener('click', function () {
        sidebar.classList.toggle('open');
    });

    // Close sidebar when clicking outside on mobile
    document.addEventListener('click', function (e) {
        if (sidebar.classList.contains('open') &&
            !sidebar.contains(e.target) &&
            e.target !== sidebarToggle) {
            sidebar.classList.remove('open');
        }
    });

    // ---- Room List (polling) ----

    function fetchRooms() {
        fetch('/api/rooms')
            .then(r => r.json())
            .then(data => {
                if (data.success && data.data) {
                    updateRoomList(data.data);
                }
            })
            .catch(() => { });
    }

    function updateRoomList(rooms) {
        roomListEl.innerHTML = '';
        rooms.forEach(room => {
            const li = document.createElement('li');
            li.textContent = room.id;
            if (room.id === currentRoom) {
                li.className = 'active';
            }
            li.addEventListener('click', () => switchRoom(room.id));
            roomListEl.appendChild(li);
        });

        // Update user count
        const current = rooms.find(r => r.id === currentRoom);
        if (current) {
            userCountEl.textContent = `${current.user_count} online`;
            headerUserCount.textContent = `${current.user_count} online`;
        }
    }

    function switchRoom(roomId) {
        if (roomId === currentRoom) return;
        currentRoom = roomId;
        disconnectWebSocket();
        messagesDiv.innerHTML = '';
        updateRoomDisplay();
        addSystemMessage(`Switched to #${currentRoom}`);
        connectWebSocket();
        sidebar.classList.remove('open');
    }

    // Poll rooms every 5 seconds
    setInterval(fetchRooms, 5000);

    // ---- Health / Fanout Check ----

    function checkHealth() {
        fetch('/api/health')
            .then(r => r.json())
            .then(data => {
                if (data.version) {
                    fanoutBadge.textContent = 'Local Mode';
                    fanoutBadge.classList.remove('active');
                }
            })
            .catch(() => { });
    }

    // ---- Keyboard Shortcuts ----

    document.addEventListener('keydown', function (e) {
        // Escape to close sidebar on mobile
        if (e.key === 'Escape') {
            sidebar.classList.remove('open');
        }
    });

})();
