const { exec, spawn } = require("child_process");
const cp = require("child_process");
const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const https = require("https");
const vm = require("vm");
const express = require("express");

const app = express();

function shell(userInput) {
  // ruleid: js-child-process-injection
  exec(`git log ${userInput}`);
  // ruleid: js-child-process-injection
  cp.execSync("rm -rf " + userInput);
  // ruleid: js-child-process-injection
  spawn(userInput, [], { shell: true });
  // ok: js-child-process-injection
  exec("ls -la");
  // ok: js-child-process-injection
  spawn("git", ["log", userInput]);
  // ok: js-child-process-injection
  /^v(\d+)/.exec(userInput);
}

function dynamic(code) {
  // ruleid: js-eval-injection
  eval(code);
  // ruleid: js-eval-injection
  new Function("a", code);
  // ruleid: js-eval-injection
  vm.runInNewContext(code, {});
  // ok: js-eval-injection
  eval("1 + 1");
}

function dom(el, html) {
  // ruleid: js-dom-xss
  el.innerHTML = html;
  // ruleid: js-dom-xss
  document.write(html);
  // ruleid: js-dom-xss
  el.insertAdjacentHTML("beforeend", html);
  // ok: js-dom-xss
  el.innerHTML = "<b>static</b>";
  // ok: js-dom-xss
  el.innerHTML = DOMPurify.sanitize(html);
  // ok: js-dom-xss
  el.textContent = html;
}

function Bio({ user }) {
  // ruleid: js-react-dangerously-set-inner-html
  return <div dangerouslySetInnerHTML={{ __html: user.bio }} />;
}

function SafeBio({ user }) {
  // ok: js-react-dangerously-set-inner-html
  return <div dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(user.bio) }} />;
}

function tls() {
  // ruleid: js-tls-verification-disabled
  https.request({ hostname: "internal", rejectUnauthorized: false });
  // ruleid: js-tls-verification-disabled
  process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
  // ok: js-tls-verification-disabled
  https.request({ hostname: "internal" });
}

async function sql(db, prisma, id) {
  // ruleid: js-sql-injection
  await db.query(`SELECT * FROM users WHERE id = ${id}`);
  // ruleid: js-sql-injection
  await db.query("SELECT * FROM users WHERE id = " + id);
  // ruleid: js-sql-injection
  await prisma.$queryRawUnsafe(`SELECT * FROM users WHERE id = ${id}`);
  // ok: js-sql-injection
  await db.query("SELECT * FROM users WHERE id = $1", [id]);
  // ok: js-sql-injection
  await prisma.$queryRaw`SELECT * FROM users WHERE id = ${id}`;
}

app.get("/files/:name", (req, res) => {
  // ruleid: js-express-path-traversal
  res.sendFile(path.join(__dirname, "uploads", req.params.name));
});

app.get("/safe/:name", (req, res) => {
  // ok: js-express-path-traversal
  res.sendFile(path.join(__dirname, "uploads", path.basename(req.params.name)));
});

app.get("/read", (req, res) => {
  // ruleid: js-express-path-traversal
  fs.readFile(req.query.file, (err, data) => res.send(data));
});

function hashes(data) {
  // ruleid: js-weak-hash
  crypto.createHash("md5").update(data);
  // ok: js-weak-hash
  crypto.createHash("sha256").update(data);
}
