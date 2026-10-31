const fs = require('fs'), path = require('path'), crypto = require('crypto');

function entry(dir) {
  const html = fs.readFileSync(path.join(dir, 'index.html'), 'utf8');
  const js = html.match(/<script\b[^>]*type="module"[^>]*src="([^"]+)"/);
  if (!js) throw new Error('Missing module entry: ' + dir);
  const css = [...html.matchAll(/<link\b[^>]*rel="stylesheet"[^>]*href="([^"]+)"/g)].map(m => m[1]);
  for (const file of [js[1], ...css]) {
    if (!file.startsWith('/assets/') || !fs.existsSync(path.join(dir, file))) throw new Error('Invalid asset: ' + file);
  }
  return { js: js[1], css };
}

function assemble(output) {
  const inputs = { lumina: 'komari-theme-Lumina/dist', console: 'komari-web/dist' };
  const entries = Object.fromEntries(Object.entries(inputs).map(([name, dir]) => [name, entry(dir)]));
  fs.mkdirSync(path.join(output, 'assets'), { recursive: true });
  function merge(sourceDir, destinationDir) {
    fs.mkdirSync(destinationDir, { recursive: true });
    for (const filename of fs.readdirSync(sourceDir)) {
      const source = path.join(sourceDir, filename), destination = path.join(destinationDir, filename);
      if (fs.statSync(source).isDirectory()) { merge(source, destination); continue; }
      const data = fs.readFileSync(source);
      if (fs.existsSync(destination) && !fs.readFileSync(destination).equals(data)) throw new Error('Asset collision: ' + filename);
      fs.writeFileSync(destination, data);
    }
  }
  for (const dir of Object.values(inputs)) merge(path.join(dir, 'assets'), path.join(output, 'assets'));
  fs.copyFileSync('komari-web/dist/favicon.ico', path.join(output, 'favicon.ico'));
  const bootstrap = fs.readFileSync('deploy/frontend-entry.js', 'utf8').replace('__KOMARI_ENTRIES__', JSON.stringify(entries));
  const filename = 'komari-app-' + crypto.createHash('sha256').update(bootstrap).digest('hex').slice(0, 16) + '.js';
  fs.writeFileSync(path.join(output, 'assets', filename), bootstrap);
  const html = fs.readFileSync('deploy/frontend-index.html', 'utf8').replace('__KOMARI_BOOTSTRAP__', '/assets/' + filename);
  fs.writeFileSync(path.join(output, 'index.html'), html);
  // Compatibility with the previous Nginx configuration; the entry is identical.
  fs.mkdirSync(path.join(output, 'control'), { recursive: true });
  fs.writeFileSync(path.join(output, 'control/index.html'), html);
  for (const [name, dir] of [['Lumina', 'komari-theme-Lumina'], ['default', 'komari-web']]) {
    const target = path.join(output, 'themes', name);
    fs.mkdirSync(target, { recursive: true });
    fs.copyFileSync(path.join(dir, 'komari-theme.json'), path.join(target, 'komari-theme.json'));
  }
  fs.writeFileSync(path.join(output, 'frontend-entries.json'), JSON.stringify(entries, null, 2));
  return entries;
}

module.exports = { assemble };
if (require.main === module) console.log(assemble(process.argv[2] || 'release/lumina-frontend'));
