const fs = require('fs'),
  tls = require('tls'),
  https = require('https'),
  http = require('http'),
  express = require('express'),
  expressWs = require('express-ws'),
  axios = require('axios');

const SERVER_NAME = 'alley-oop.example.com';
const SERVER_USERNAME = 'alley-oop';
const SERVER_PASSWORD = 'password';
const HTTP_PORT = 1080; // to avoid need for sudo
const HTTPS_PORT = 10443; // ^ ditto

startServer({
  'test1.lan.example.com': '10.6.3.8',
});

function fetchUrl(url) {
  console.log(`Fetching ${url}`);
  return axios.get(url, { auth: { username: SERVER_USERNAME, password: SERVER_PASSWORD } }).then(res => res.data);
}

function readFile(path) {
  return new Promise((resolve, reject) => {
    fs.readFile(path, (err, data) => {
      if (err) reject(err);
      else resolve(data);
    });
  });
}

function writeFile(path, data) {
  return new Promise((resolve, reject) => {
    fs.writeFile(path, data, err => {
      if (err) reject(err);
      else resolve(data);
    });
  });
}

function fetchWithCache(url, path) {
  return readFile(path)
    .catch(err =>
      err.code === 'ENOENT' // file not found
        ? fetchUrl(url).then(data => writeFile(path, data))
        : Promise.reject(err),
    );
}

function getKeyPair(hostname) {
  return Promise.all([
    fetchWithCache(
      `https://${SERVER_NAME}/v1/privatekey?hostname=${hostname}`,
      `./cache/${hostname}.key`,
    ),
    fetchWithCache(
      `https://${SERVER_NAME}/v1/certificate?hostname=${hostname}`,
      `./cache/${hostname}.crt`,
    ),
  ]);
}

function updateDns(hostname, ip) {
  return fetchUrl(`https://${SERVER_NAME}/v1/update?hostname=${hostname}&myip=${ip}`);
}

function getHttpsOptions() {
  const secureContext = {};
  return {
    key: fs.readFileSync(__dirname + '/dummy.key', 'utf8'), // https://msol.io/blog/tech/create-a-self-signed-ecc-certificate/
    cert: fs.readFileSync(__dirname + '/dummy.crt', 'utf8'),
    SNICallback: (domain, cb) => {
      if (secureContext[domain]) {
        cb(null, secureContext[domain]);
      } else {
        getKeyPair(domain).then(
          ([ key, cert ]) => {
            secureContext[domain] = tls.createSecureContext({ key, cert }); // https://stackoverflow.com/a/38862651
            cb(null, secureContext[domain]);
          },
          err => console.log('SNI Error:', err.message, err.config && err.config.url),
        );
      }
    },
  };
}

function startServer(domainMap) {
  return Promise.resolve()
    .then(() => Promise.all(
      Object.keys(domainMap)
        .map(domain => updateDns(domain, domainMap[domain]))
    ))
    .then(() => {
      var app = express();
      const httpServer = http.createServer(app);
      const httpsServer = https.createServer(getHttpsOptions(), app);
      httpServer.listen(HTTP_PORT);
      httpsServer.listen(HTTPS_PORT);
      expressWs(app, httpServer);
      expressWs(app, httpsServer);
      app.get('/', (req, res) => res.send(`Hello "${req.hostname}" over ${req.protocol.toUpperCase()}!`));
      app.ws('/ws', (ws, req) => ws.send(`Hello "${req.hostname}" over WS!`));
    })
    .then(() => {
      console.log('\nServer started! Try any of the following endpoints:\n');
      Object.keys(domainMap).forEach(domain => {
        console.log([
          `* http://${domain}:${HTTP_PORT}/`,
          `* https://${domain}:${HTTPS_PORT}/`,
          `* ws://${domain}:${HTTP_PORT}/ws (test with http://www.websocket.org/echo.html)`,
          `* wss://${domain}:${HTTPS_PORT}/ws (test with https://www.websocket.org/echo.html)`,
          '',
        ].join('\n'));
      });
    })
    .catch(err => console.log('Error:', err.message, err.config && err.config.url))
}
