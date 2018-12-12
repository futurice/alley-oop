const os = require('os'),
  tls = require('tls'),
  https = require('https'),
  http = require('http'),
  express = require('express'),
  expressWs = require('express-ws'),
  axios = require('axios');

const config = {
  DOMAIN_NAME: process.env.DOMAIN_NAME || '',
  SERVER_NAME: process.env.SERVER_NAME || '',
  SERVER_USERNAME: process.env.SERVER_USERNAME || 'alley-oop',
  SERVER_PASSWORD: process.env.SERVER_PASSWORD || '',
  HTTP_PORT: process.env.HTTP_PORT || 1080, // to avoid need for sudo (otherwise can use 80)
  HTTPS_PORT: process.env.HTTPS_PORT || 10443, // ^ ditto (otherwise can use 443)
}

if (checkConfig()) {
  startServer(getDefaultDomainMap());
}

// Starts a HTTP+HTTPS+WS+WSS demo server, mapping the given domain names to the given IP's.
// @example startServer({ 'test.lan.example.com': '192.168.1.123' })
function startServer(domainMap) {
  return Promise.resolve()
    .then(() =>
      Promise.all(
        Object.keys(domainMap).map(domain =>
          updateDns(domain, domainMap[domain])
        )
      )
    )
    .then(() => {
      var app = express();
      const httpServer = http.createServer(app);
      const httpsServer = https.createServer(getHttpsOptions(), app);
      httpServer.listen(config.HTTP_PORT);
      httpsServer.listen(config.HTTPS_PORT);
      expressWs(app, httpServer);
      expressWs(app, httpsServer);
      app.get('/', (req, res) =>
        res.send(`Hello "${req.hostname}" over ${req.protocol.toUpperCase()}!`)
      );
      app.ws('/ws', (ws, req) => ws.send(`Hello "${req.hostname}" over WS!`));
    })
    .then(() => {
      console.log('\nServer started! Try any of the following endpoints:\n');
      Object.keys(domainMap).forEach(logDomainInfo);
    })
    .catch(err =>
      console.log('Error:', err.message, err.config && err.config.url)
    );
}

// @see https://en.wikipedia.org/wiki/Private_network#Private_IPv4_address_spaces
function isPrivateIpV4Address(ip) {
  if (!ip.match(/^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4}$/)) return false;
  if (ip.match(/^10\./)) return true; // 10.0.0.0 - 10.255.255.255
  if (ip.match(/^172\.(?:1[6789]|2[0123456789]|3[01])\./)) return true; // 172.16.0.0 - 172.31.255.255
  if (ip.match(/^192\.168\./)) return true; // 192.168.0.0 - 192.168.255.255
  return false;
}

// Queries the network interfaces of the host OS, and picks ones with local-looking IP's.
// Then, creates dyn-domains for each.
// @example { '192-168-1-123.lan.example.com': '192.168.1.123' }
function getDefaultDomainMap() {
  const domain = ip => `${ip.replace(/\./g, '-')}.${config.DOMAIN_NAME}`;
  return Object.values(os.networkInterfaces())
    .reduce((memo, next) => memo.concat(next), [])
    .map(net => net.address)
    .filter(isPrivateIpV4Address)
    .reduce((memo, next) => ({ ...memo, [domain(next)]: next }), {});
}

// Logs the current config, so the user knows how to override the settings if need be.
function checkConfig() {
  console.log(`Current configuration is:\n`);
  console.log(config);
  console.log();
  if (config.DOMAIN_NAME && config.SERVER_NAME && config.SERVER_PASSWORD) {
    return true;
  } else {
    console.log(`ERROR: Required configuration is missing.`)
    console.log(`\nYou can set it via environment variables, for example:\n`)
    console.log(`  $ DOMAIN_NAME=lan.example.com SERVER_NAME=alley-oop.example.com SERVER_PASSWORD=password npm start\n`);
  }
}

// Logs helpful information about connecting to the given domain.
function logDomainInfo(domain) {
  console.log(
    [
      `* http://${domain}:${config.HTTP_PORT}/`,
      `* https://${domain}:${config.HTTPS_PORT}/`,
      `* ws://${domain}:${config.HTTP_PORT}/ws (test with http://www.websocket.org/echo.html)`,
      `* wss://${domain}:${config.HTTPS_PORT}/ws (test with https://www.websocket.org/echo.html)`,
      ``
    ].join('\n')
  );
}

// Convenience wrapper for axios.get().
function fetch(url) {
  console.log(`Fetching ${url}`);
  return axios.get(url, { auth: { username: config.SERVER_USERNAME, password: config.SERVER_PASSWORD } })
    .then(res => res.data);
}

// Contacts the alley-oop server for a key pair for the given hostname.
function getKeyPair(hostname) {
  return Promise.all([
    fetch(`https://${config.SERVER_NAME}/v1/privatekey?hostname=${hostname}`),
    fetch(`https://${config.SERVER_NAME}/v1/certificate?hostname=${hostname}`),
  ]);
}

// Asks the alley-oop server to point the given hostname to the given IP.
function updateDns(hostname, ip) {
  return fetch(`https://${config.SERVER_NAME}/v1/update?hostname=${hostname}&myip=${ip}`);
}

// Returns the configuration object for https.createServer().
// The SNICallback is not invoked for every request, only when needed.
function getHttpsOptions() {
  return {
    ...getDummyKeyPair(),
    SNICallback: (domain, cb) => {
      getKeyPair(domain).then(
        ([key, cert]) => cb(null, tls.createSecureContext({ key, cert })), // https://stackoverflow.com/a/38862651
        err => console.log('SNI Error:', err.message, err.config && err.config.url),
      );
    }
  };
}

// Even when you provide an SNICallback for giving node's https server a certificate per domain, it still needs a valid key pair for startup.
// Thus, we have a self-signed certificate for "example.com", valid for 10 years, whose private key is perfectly fine to commit to version control.
// No sane client will ever accept it for any connection, and they're not supposed to.
// @see https://msol.io/blog/tech/create-a-self-signed-ecc-certificate/
function getDummyKeyPair() {
  return {
    key: '-----BEGIN EC PARAMETERS-----\nBggqhkjOPQMBBw==\n-----END EC PARAMETERS-----\n-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIInuZa6NQE4sQAJ1M2VkFrakgXiC7tp+UcFJNrXg9y4+oAoGCCqGSM49\nAwEHoUQDQgAEjvk5okzcKsdY5iUWy0Y7WI50DljlxzUMmtYklGRiLRlCGL0C3n+U\nbxEF8j5sBxnxtbM8Bi47gwyGgIvd6S6EYw==\n-----END EC PRIVATE KEY-----\n',
    cert: '-----BEGIN CERTIFICATE-----\nMIIBmDCCAT+gAwIBAgIJAOieWDdQkniGMAkGByqGSM49BAEwFjEUMBIGA1UEAxML\nZXhhbXBsZS5jb20wHhcNMTgwMzI3MTEwNDE0WhcNMjgwMzI0MTEwNDE0WjAWMRQw\nEgYDVQQDEwtleGFtcGxlLmNvbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABI75\nOaJM3CrHWOYlFstGO1iOdA5Y5cc1DJrWJJRkYi0ZQhi9At5/lG8RBfI+bAcZ8bWz\nPAYuO4MMhoCL3ekuhGOjdzB1MB0GA1UdDgQWBBSoWjhsol9/EKFbZWPVnjr8BEtI\nCzBGBgNVHSMEPzA9gBSoWjhsol9/EKFbZWPVnjr8BEtIC6EapBgwFjEUMBIGA1UE\nAxMLZXhhbXBsZS5jb22CCQDonlg3UJJ4hjAMBgNVHRMEBTADAQH/MAkGByqGSM49\nBAEDSAAwRQIhAPo/a+shFimxpD6pS/yUIbjTKDgV/f4NBy7BBpetgj3VAiBkp823\n0qgVnYJCqWXX/bhzORPVtEAFWDsMY4aEWusWog==\n-----END CERTIFICATE-----\n'
  };
}

