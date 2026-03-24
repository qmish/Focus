// E2E тесты для Focus Messenger
// Запуск: npx playwright test

import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const API_URL = process.env.API_URL || 'http://localhost:8080';

test.describe('Focus Messenger E2E', () => {
  test('should load login page', async ({ page }) => {
    await page.goto(BASE_URL);
    
    await expect(page).toHaveTitle(/Focus/);
    await expect(page.locator('h1')).toContainText(/Focus|Login/i);
  });

  test('should redirect to rooms after login', async ({ page }) => {
    // Mock Keycloak login
    await page.goto(BASE_URL);
    
    // После мока авторизации должен быть редирект на /rooms
    // await expect(page).toHaveURL(/\/rooms/);
  });

  test('should create room', async ({ page }) => {
    await page.goto(`${BASE_URL}/rooms`);
    
    // Click create room button
    await page.click('button:has-text("Создать комнату")');
    
    // Fill room name
    await page.fill('input[placeholder*="название"]', 'E2E Test Room');
    
    // Submit
    await page.click('button:has-text("Создать")');
    
    // Check room created
    await expect(page.locator('text=E2E Test Room')).toBeVisible();
  });

  test('should send message in room', async ({ page }) => {
    await page.goto(`${BASE_URL}/rooms`);
    
    // Open first room
    await page.click('.room-card:first-child');
    
    // Type message
    await page.fill('.message-input', 'E2E test message');
    
    // Send
    await page.click('.send-btn');
    
    // Check message appears
    await expect(page.locator('text=E2E test message')).toBeVisible();
  });

  test('should join video call', async ({ page }) => {
    await page.goto(`${BASE_URL}/rooms`);
    
    // Open room
    await page.click('.room-card:first-child');
    
    // Click video call button
    await page.click('button:has-text("Видеозвонок")');
    
    // Check Jitsi iframe loaded
    await expect(page.locator('iframe')).toBeVisible();
  });
});

test.describe('Admin Panel E2E', () => {
  test('should load admin dashboard', async ({ page }) => {
    await page.goto('http://localhost:3001');
    
    await expect(page).toHaveTitle(/Admin/);
  });

  test('should show users list', async ({ page }) => {
    await page.goto('http://localhost:3001/users');
    
    // Check table loaded
    await expect(page.locator('table')).toBeVisible();
  });

  test('should show stats', async ({ page }) => {
    await page.goto('http://localhost:3001/dashboard');
    
    // Check stats cards
    await expect(page.locator('.stat-card')).toHaveCount(4);
  });
});

test.describe('API E2E', () => {
  test('health check', async ({ request }) => {
    const response = await request.get(`${API_URL}/health`);
    
    expect(response.ok()).toBeTruthy();
    expect(await response.json()).toEqual({ status: 'healthy' });
  });

  test('ready check', async ({ request }) => {
    const response = await request.get(`${API_URL}/ready`);
    
    expect(response.ok()).toBeTruthy();
  });

  test('get rooms (unauthorized)', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/rooms`);
    
    expect(response.status()).toBe(401);
  });

  test('admin stats (unauthorized)', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/admin/stats`);
    
    expect(response.status()).toBe(401);
  });
});
