import hashlib
import os
import pickle
import ssl
import subprocess
import tarfile
import tempfile

import jinja2
import requests
import yaml
from flask import Flask, request, send_file

app = Flask(__name__)


def shell(user_input):
    # ruleid: python-shell-injection
    subprocess.run(f"git log {user_input}", shell=True)
    # ruleid: python-shell-injection
    os.system("rm -rf " + user_input)
    # ok: python-shell-injection
    subprocess.run("ls -la", shell=True)
    # ok: python-shell-injection
    subprocess.run(["git", "log", user_input])


def dynamic(code):
    # ruleid: python-eval-injection
    eval(code)
    # ruleid: python-eval-injection
    exec(code)
    # ok: python-eval-injection
    eval("1 + 1")


def deserialize(blob, path):
    # ruleid: python-insecure-deserialization
    pickle.loads(blob)
    with open(path, "rb") as f:
        # ruleid: python-insecure-deserialization
        pickle.load(f)


def load_yaml(text):
    # ruleid: python-unsafe-yaml-load
    yaml.load(text)
    # ruleid: python-unsafe-yaml-load
    yaml.load(text, Loader=yaml.Loader)
    # ok: python-unsafe-yaml-load
    yaml.load(text, Loader=yaml.SafeLoader)
    # ok: python-unsafe-yaml-load
    yaml.safe_load(text)


def sql(cursor, name):
    # ruleid: python-sql-injection
    cursor.execute(f"SELECT * FROM users WHERE name = '{name}'")
    # ruleid: python-sql-injection
    cursor.execute("SELECT * FROM users WHERE name = '%s'" % name)
    # ruleid: python-sql-injection
    cursor.execute("SELECT * FROM users WHERE name = '{}'".format(name))
    query = f"DELETE FROM users WHERE name = '{name}'"
    # ruleid: python-sql-injection
    cursor.execute(query)
    # ok: python-sql-injection
    cursor.execute("SELECT * FROM users WHERE name = %s", (name,))


def tls():
    # ruleid: python-tls-verification-disabled
    requests.get("https://internal.example", verify=False, timeout=5)
    # ruleid: python-tls-verification-disabled
    ssl._create_unverified_context()
    # ok: python-tls-verification-disabled
    requests.get("https://internal.example", timeout=5)


def timeouts():
    # ruleid: python-request-without-timeout
    requests.get("https://example.com")
    # ok: python-request-without-timeout
    requests.post("https://example.com", json={}, timeout=10)


def hashes(data):
    # ruleid: python-weak-hash
    hashlib.md5(data)
    # ok: python-weak-hash
    hashlib.md5(data, usedforsecurity=False)
    # ok: python-weak-hash
    hashlib.sha256(data)


def tmp():
    # ruleid: python-insecure-tempfile
    tempfile.mktemp()
    # ok: python-insecure-tempfile
    tempfile.mkstemp()


def extract(path, dest):
    with tarfile.open(path) as tar:
        # ruleid: python-tarfile-extractall
        tar.extractall(dest)
    with tarfile.open(path) as tar:
        # ok: python-tarfile-extractall
        tar.extractall(dest, filter="data")


def templates():
    # ruleid: python-jinja2-autoescape-disabled
    jinja2.Environment(autoescape=False)
    # ok: python-jinja2-autoescape-disabled
    jinja2.Environment(autoescape=True)


@app.route("/ping")
def ping():
    host = request.args.get("host")
    # ruleid: python-request-input-to-command
    subprocess.run(["ping", "-c", "1", host])
    # ok: python-request-input-to-command
    subprocess.run(["ping", "-c", "1", "localhost"])
    return "ok"


@app.route("/download")
def download():
    name = request.args.get("file")
    # ruleid: python-request-input-to-file-path
    return send_file(os.path.join("/srv/files", name))


@app.route("/download-safe")
def download_safe():
    name = request.args.get("file")
    # ok: python-request-input-to-file-path
    return send_file(os.path.join("/srv/files", os.path.basename(name)))


@app.route("/fetch")
def fetch():
    url = request.args.get("url")
    # ruleid: python-request-input-to-ssrf
    requests.get(url, timeout=5)
    # ok: python-request-input-to-ssrf
    requests.get("https://example.com/status", timeout=5)
    return "ok"


if __name__ == "__main__":
    # ruleid: python-flask-debug-enabled
    app.run(debug=True)
