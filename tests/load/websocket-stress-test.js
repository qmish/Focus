// Стресс тест для WebSocket
// Запуск: k6 run tests/load/websocket-stress-test.js

import ws from 'k6/ws';
import { check } from 'k6';

const WS_URL = __ENV.WS_URL || 'ws://localhost:8080/api/v1/ws';

export const options = {
  stages: [
    { duration: '30s', target: 5 },
    { duration: '1m', target: 20 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    ws_connecting: ['p(95)<100'],
    ws_session_duration: ['p(95)<10000'],
  },
};

export default function () {
  const response = ws.connect(WS_URL, {}, function (socket) {
    socket.on('open', () => {
      console.log('WebSocket connected');
      
      // Subscribe to room
      socket.send(JSON.stringify({
        type: 'subscribe',
        payload: { room_id: 'test-room' },
      }));
    });
    
    socket.on('message', (data) => {
      console.log('Message received:', data);
    });
    
    socket.on('close', () => {
      console.log('WebSocket disconnected');
    });
    
    socket.on('error', (error) => {
      console.log('WebSocket error:', error);
    });
    
    socket.setTimeout(() => {
      socket.close();
    }, 60000);
  });
  
  check(response, {
    'status is 101': (r) => r && r.status === 101,
  });
}
