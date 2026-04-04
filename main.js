const { app, BrowserWindow, ipcMain, shell, Menu } = require('electron');
const path = require('path');
const fs = require('fs');

app.commandLine.appendSwitch('disable-gpu-shader-disk-cache');

const isDev = process.env.NODE_ENV !== 'production';
const configPath = path.join(app.getPath('userData'), 'config.json');

// Default config
let config = {
  data: { selectedTeams: [] },
  aesthetics: { themeColor: '#1a5fa8', bubbleDarkMode: false }
};

try {
  if (fs.existsSync(configPath)) {
    config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
    // Migrate old boolean-flag format to selectedTeams array
    if (config.data && !Array.isArray(config.data.selectedTeams)) {
      config.data = { selectedTeams: [] };
      fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
    }
  } else {
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
  }
} catch (e) {
  console.error("Failed to load config", e);
}

let mainWindow = null;
let settingsWindow = null;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 280,
    height: 360,
    transparent: true,
    frame: false,
    resizable: false,
    alwaysOnTop: true,
    skipTaskbar: true,
    hasShadow: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  mainWindow.setMenuBarVisibility(false);

  if (isDev) {
    mainWindow.loadURL('http://localhost:51234/#/chibi');
  } else {
    mainWindow.loadFile(path.join(__dirname, 'dist', 'index.html'), { hash: '/chibi' });
  }
}

function createSettingsWindow() {
  if (settingsWindow) {
    settingsWindow.focus();
    return;
  }

  settingsWindow = new BrowserWindow({
    width: 400,
    height: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  settingsWindow.setMenuBarVisibility(false);

  if (isDev) {
    settingsWindow.loadURL('http://localhost:51234/#/settings');
  } else {
    settingsWindow.loadFile(path.join(__dirname, 'dist', 'index.html'), { hash: '/settings' });
  }

  settingsWindow.on('closed', () => {
    settingsWindow = null;
  });
}

app.whenReady().then(() => {
  ipcMain.on('open-external', (event, url) => {
    if (url && (url.startsWith('http://') || url.startsWith('https://'))) {
      shell.openExternal(url);
    }
  });

  ipcMain.on('show-context-menu', (event) => {
    const template = [
      {
        label: 'Settings',
        click: () => { createSettingsWindow(); }
      },
      { type: 'separator' },
      { label: 'Quit', role: 'quit' }
    ];
    const menu = Menu.buildFromTemplate(template);
    menu.popup(BrowserWindow.fromWebContents(event.sender));
  });

  ipcMain.on('save-settings', (event, newConfig) => {
    config = newConfig;
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
    if (mainWindow) {
      mainWindow.webContents.send('settings-saved', config);
    }

    // Sync selected teams to Go backend
    const http = require('http');
    const payload = JSON.stringify({ selectedTeams: config.data?.selectedTeams || [] });
    const req = http.request('http://localhost:8080/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(payload) }
    });
    req.on('error', (err) => console.error('Failed to sync config to backend:', err.message));
    req.write(payload);
    req.end();
  });

  ipcMain.handle('get-settings', () => {
    return config;
  });

  ipcMain.handle('send-query', async (event, query) => {
    const http = require('http');
    const payload = JSON.stringify({ query });

    return new Promise((resolve, reject) => {
      const req = http.request('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(payload) }
      }, (res) => {
        let body = '';
        res.on('data', (chunk) => { body += chunk; });
        res.on('end', () => {
          try {
            resolve(JSON.parse(body));
          } catch {
            reject(new Error('Invalid response from backend'));
          }
        });
      });
      req.on('error', (err) => reject(err));
      req.write(payload);
      req.end();
    });
  });

  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});
