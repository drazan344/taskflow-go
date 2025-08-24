const WebSocket = require('ws');

// JWT token from login
const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNDBjYTdiMDUtMmU5MC00YmYyLWJmZDMtNGRjZDAwODAwZWFiIiwidGVuYW50X2lkIjoiZmE0NDg0NjgtYTgyMC00OTEwLWExNmYtYzkzYTEwMDM5NmY0IiwiZW1haWwiOiJhZG1pbkBuZXdjb21wYW55LmNvbSIsInJvbGUiOiJhZG1pbiIsInRva2VuX3R5cGUiOiJhY2Nlc3MiLCJzZXNzaW9uX2lkIjoiNTYwOWIzODQtZTYwMy00MjA3LTgxYTMtM2IyNThmODM3MTdkIiwiaXNzIjoidGFza2Zsb3ctZ28iLCJzdWIiOiI0MGNhN2IwNS0yZTkwLTRiZjItYmZkMy00ZGNkMDA4MDBlYWIiLCJleHAiOjE3NTU5NDYzMTcsIm5iZiI6MTc1NTg1OTkxNywiaWF0IjoxNzU1ODU5OTE3LCJqdGkiOiI3MjkxYmQ0Yi1iYTkzLTQzMTctYWNhNy1jNzE1OGVmODUzMTUifQ.7msJLbsMafnj-6K7hjv1jldLD7M1Xn23A6FW5u-0eTQ";

console.log('🚀 Testing TaskFlow WebSocket Connection...');

// Create WebSocket connection
const ws = new WebSocket('ws://localhost:8080/api/v1/ws', {
    headers: {
        'Authorization': `Bearer ${token}`
    }
});

ws.on('open', function open() {
    console.log('✅ Connected to TaskFlow WebSocket!');
    
    // Send a ping message
    const pingMessage = {
        type: 'ping',
        timestamp: Date.now()
    };
    
    ws.send(JSON.stringify(pingMessage));
    console.log('📤 Sent ping message');
    
    // Send a test message
    setTimeout(() => {
        const testMessage = {
            type: 'custom_message',
            data: {
                text: 'Hello from Node.js WebSocket test client!',
                timestamp: new Date().toISOString()
            }
        };
        
        ws.send(JSON.stringify(testMessage));
        console.log('📤 Sent test message');
    }, 1000);
});

ws.on('message', function incoming(data) {
    try {
        const message = JSON.parse(data);
        console.log('📨 Received message:', message);
        
        if (message.type === 'pong') {
            console.log('🏓 Pong received - WebSocket is working!');
        }
    } catch (e) {
        console.log('📨 Raw message:', data.toString());
    }
});

ws.on('close', function close() {
    console.log('❌ WebSocket connection closed');
});

ws.on('error', function error(err) {
    console.error('❌ WebSocket error:', err);
});

// Keep the connection alive for testing
setTimeout(() => {
    console.log('🏁 Test completed, closing connection...');
    ws.close();
    process.exit(0);
}, 5000);