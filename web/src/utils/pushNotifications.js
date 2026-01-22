// Push notification utilities

// Check if push notifications are supported
export function isPushSupported() {
  return 'serviceWorker' in navigator && 'PushManager' in window;
}

// Check current notification permission
export function getNotificationPermission() {
  if (!('Notification' in window)) return 'unsupported';
  return Notification.permission; // 'granted', 'denied', or 'default'
}

// Request notification permission
export async function requestNotificationPermission() {
  if (!('Notification' in window)) {
    throw new Error('Notifications not supported');
  }
  return await Notification.requestPermission();
}

// Register service worker
export async function registerServiceWorker() {
  if (!('serviceWorker' in navigator)) {
    throw new Error('Service workers not supported');
  }

  try {
    const registration = await navigator.serviceWorker.register('/sw.js');
    console.log('Service worker registered:', registration.scope);
    return registration;
  } catch (error) {
    console.error('Service worker registration failed:', error);
    throw error;
  }
}

// Get existing service worker registration
export async function getServiceWorkerRegistration() {
  if (!('serviceWorker' in navigator)) {
    return null;
  }
  return await navigator.serviceWorker.getRegistration();
}

// Get VAPID public key from server
export async function getVAPIDPublicKey() {
  const response = await fetch('/api/vapid-public-key');
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to get VAPID key');
  }
  const data = await response.json();
  return data.publicKey;
}

// Convert VAPID key to Uint8Array
function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, '+')
    .replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

// Subscribe to push notifications
export async function subscribeToPush(registration, vapidPublicKey) {
  const applicationServerKey = urlBase64ToUint8Array(vapidPublicKey);

  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey
  });

  // Send subscription to server
  const response = await fetch('/api/subscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ subscription: JSON.stringify(subscription) })
  });

  if (!response.ok) {
    throw new Error('Failed to save subscription on server');
  }

  console.log('Push subscription saved:', subscription);
  return subscription;
}

// Unsubscribe from push notifications
export async function unsubscribeFromPush(registration) {
  const subscription = await registration.pushManager.getSubscription();

  if (subscription) {
    await subscription.unsubscribe();
    console.log('Push subscription removed locally');
  }

  // Notify server
  const response = await fetch('/api/unsubscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' }
  });

  if (!response.ok) {
    console.warn('Failed to unsubscribe on server');
  }

  return true;
}

// Check if user is subscribed to push
export async function isSubscribedToPush(registration) {
  if (!registration) return false;
  const subscription = await registration.pushManager.getSubscription();
  return !!subscription;
}

// Full setup flow: register SW, request permission, subscribe
export async function setupPushNotifications() {
  if (!isPushSupported()) {
    throw new Error('Push notifications not supported');
  }

  // Request permission
  const permission = await requestNotificationPermission();
  if (permission !== 'granted') {
    throw new Error('Notification permission denied');
  }

  // Register service worker
  const registration = await registerServiceWorker();

  // Wait for service worker to be ready
  await navigator.serviceWorker.ready;

  // Get VAPID key
  const vapidKey = await getVAPIDPublicKey();
  if (!vapidKey) {
    throw new Error('VAPID keys not configured on server');
  }

  // Subscribe
  const subscription = await subscribeToPush(registration, vapidKey);
  return { registration, subscription };
}

// Test notification (local, doesn't go through server)
export async function sendTestNotification() {
  if (Notification.permission !== 'granted') {
    throw new Error('Notification permission not granted');
  }

  const registration = await getServiceWorkerRegistration();
  if (!registration) {
    throw new Error('Service worker not registered');
  }

  await registration.showNotification('LineFinder Test', {
    body: 'Push notifications are working!',
    icon: '/icon-192.png',
    badge: '/badge-72.png',
    tag: 'test'
  });
}
