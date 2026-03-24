// Нагрузочные тесты для Focus API
// Запуск: k6 run tests/load/api-load-test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Кастомные метрики
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency');

// Конфигурация
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_USER_EMAIL = __ENV.TEST_USER_EMAIL || 'test@example.com';
const TEST_USER_PASSWORD = __ENV.TEST_USER_PASSWORD || 'password';

export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Разогрев до 10 пользователей
    { duration: '1m', target: 50 },    // Нагрузка до 50 пользователей
    { duration: '2m', target: 100 },   // Пиковая нагрузка 100 пользователей
    { duration: '1m', target: 50 },    // Снижение до 50
    { duration: '30s', target: 0 },    // Завершение
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% запросов < 500ms
    http_req_failed: ['rate<0.01'],    // Менее 1% ошибок
    errors: ['rate<0.1'],              // Менее 10% ошибок
    api_latency: ['p(95)<300'],        // API latency < 300ms
  },
};

// Health check тест
export function healthCheck() {
  const res = http.get(`${BASE_URL}/health`);
  
  const checkResult = check(res, {
    'health status is 200': (r) => r.status === 200,
    'health response is healthy': (r) => r.json().status === 'healthy',
  });
  
  errorRate.add(!checkResult);
  apiLatency.add(res.timings.duration);
  
  return checkResult;
}

// Auth тест
export function authTest() {
  // Тест login endpoint
  const loginRes = http.get(`${BASE_URL}/api/v1/auth/login`, {
    redirects: 0, // Не следовать редиректам
  });
  
  const checkResult = check(loginRes, {
    'login redirects to keycloak': (r) => r.status === 302,
  });
  
  errorRate.add(!checkResult);
  
  return checkResult;
}

// Rooms CRUD тест
export function roomsTest(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  // GET /api/v1/rooms
  const getRes = http.get(`${BASE_URL}/api/v1/rooms`, { headers });
  check(getRes, {
    'get rooms status is 200': (r) => r.status === 200,
  });
  
  // POST /api/v1/rooms
  const createPayload = JSON.stringify({
    name: `Test Room ${Date.now()}`,
    type: 'public',
  });
  const postRes = http.post(`${BASE_URL}/api/v1/rooms`, createPayload, { headers });
  check(postRes, {
    'create room status is 201': (r) => r.status === 201,
  });
  
  const room = postRes.json();
  
  if (room.id) {
    // GET /api/v1/rooms/:id
    const getOneRes = http.get(`${BASE_URL}/api/v1/rooms/${room.id}`, { headers });
    check(getOneRes, {
      'get room status is 200': (r) => r.status === 200,
    });
    
    // DELETE /api/v1/rooms/:id
    const deleteRes = http.del(`${BASE_URL}/api/v1/rooms/${room.id}`, null, { headers });
    check(deleteRes, {
      'delete room status is 204': (r) => r.status === 204,
    });
  }
}

// Messages тест
export function messagesTest(token, roomId) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  // GET /api/v1/messages
  const getRes = http.get(`${BASE_URL}/api/v1/messages?room_id=${roomId}`, { headers });
  check(getRes, {
    'get messages status is 200': (r) => r.status === 200,
  });
  
  // POST /api/v1/messages
  const createPayload = JSON.stringify({
    room_id: roomId,
    content: `Test message ${Date.now()}`,
    type: 'text',
  });
  const postRes = http.post(`${BASE_URL}/api/v1/messages`, createPayload, { headers });
  check(postRes, {
    'create message status is 201': (r) => r.status === 201,
  });
}

// Admin тест
export function adminTest(adminToken) {
  const headers = {
    'Authorization': `Bearer ${adminToken}`,
  };
  
  // GET /api/v1/admin/users
  const usersRes = http.get(`${BASE_URL}/api/v1/admin/users`, { headers });
  check(usersRes, {
    'get users status is 200': (r) => r.status === 200,
  });
  
  // GET /api/v1/admin/stats
  const statsRes = http.get(`${BASE_URL}/api/v1/admin/stats`, { headers });
  check(statsRes, {
    'get stats status is 200': (r) => r.status === 200,
  });
  
  // GET /api/v1/admin/conferences
  const confRes = http.get(`${BASE_URL}/api/v1/admin/conferences`, { headers });
  check(confRes, {
    'get conferences status is 200': (r) => r.status === 200,
  });
}

// Главный сценарий
export default function () {
  // Health check
  healthCheck();
  sleep(1);
  
  // Auth тест (без токена)
  authTest();
  sleep(1);
  
  // Rooms тест (с заглушкой токена)
  const mockToken = 'mock-token-for-load-test';
  roomsTest(mockToken);
  sleep(1);
  
  // Messages тест
  const mockRoomId = '00000000-0000-0000-0000-000000000000';
  messagesTest(mockToken, mockRoomId);
  sleep(1);
  
  // Admin тест
  adminTest(mockToken);
  sleep(1);
}

// Summary для отчёта
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'tests/load/results.json': JSON.stringify(data),
    'tests/load/results.html': htmlReport(data),
  };
}

function textSummary(data, options) {
  return `
╔═══════════════════════════════════════════════════════════╗
║          Focus API Load Test Summary                      ║
╠═══════════════════════════════════════════════════════════╣
║  Total Requests:     ${data.metrics.http_reqs.values.count.toString().padStart(10)}                     ║
║  Request Duration:   ${data.metrics.http_req_duration.values.avg.toFixed(2).padStart(10)} ms (avg)            ║
║  P(95) Duration:     ${data.metrics.http_req_duration.values['p(95)'].toFixed(2).padStart(10)} ms                   ║
║  Failed Requests:    ${data.metrics.http_req_failed.values.rate.toFixed(4).padStart(10)}                      ║
╚═══════════════════════════════════════════════════════════╝
`;
}

function htmlReport(data) {
  return `<!DOCTYPE html>
<html>
<head><title>Focus API Load Test Report</title></head>
<body>
  <h1>Load Test Report</h1>
  <p>Generated: ${new Date().toISOString()}</p>
  <h2>Summary</h2>
  <ul>
    <li>Total Requests: ${data.metrics.http_reqs.values.count}</li>
    <li>Avg Duration: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms</li>
    <li>P95 Duration: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms</li>
    <li>Failed Rate: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%</li>
  </ul>
</body>
</html>`;
}
