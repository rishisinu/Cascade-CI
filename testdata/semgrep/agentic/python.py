import os
import sqlite3
import subprocess

import anthropic
import requests
import torch
from flask import Flask, request
from langchain_community.vectorstores import FAISS
from langchain_core.tools import tool
from langchain_experimental.tools import PythonREPLTool
from mcp.server.fastmcp import FastMCP
from transformers import AutoModelForCausalLM

client = anthropic.Anthropic()
mcp = FastMCP("demo")
app = Flask(__name__)


def run_generated_code(task):
    resp = client.messages.create(
        model="claude-opus-5-5",
        max_tokens=1024,
        messages=[{"role": "user", "content": task}],
    )
    code = resp.content[0].text
    # ruleid: agentic-llm-output-to-code-execution
    exec(code)


def run_generated_command(task):
    resp = client.messages.create(model="claude-opus-5-5", max_tokens=256, messages=[{"role": "user", "content": task}])
    cmd = resp.content[0].text.strip()
    # ruleid: agentic-llm-output-to-code-execution
    subprocess.run(cmd, shell=True)
    # ok: agentic-llm-output-to-code-execution
    subprocess.run(["echo", "done"])


def generated_sql(conn: sqlite3.Connection, question):
    resp = client.messages.create(model="claude-opus-5-5", max_tokens=256, messages=[{"role": "user", "content": question}])
    query = resp.content[0].text
    # ruleid: agentic-llm-output-to-sql
    conn.execute(query)
    # ok: agentic-llm-output-to-sql
    conn.execute("SELECT name FROM products WHERE id = ?", (query,))


@mcp.tool()
def run_shell(command: str) -> str:
    # ruleid: agentic-tool-input-to-dangerous-sink
    return subprocess.check_output(command, shell=True, text=True)


@mcp.tool()
def read_note(name: str) -> str:
    # ruleid: agentic-tool-input-to-dangerous-sink
    with open(os.path.join("notes", name)) as f:
        return f.read()


@mcp.tool()
def read_note_safe(name: str) -> str:
    # ok: agentic-tool-input-to-dangerous-sink
    with open(os.path.join("notes", os.path.basename(name))) as f:
        return f.read()


@tool
def lookup(term: str) -> str:
    """Look up a term."""
    # ok: agentic-tool-input-to-dangerous-sink
    return requests.get("https://api.example.com/lookup", params={"q": term}, timeout=5).text


@app.route("/chat", methods=["POST"])
def chat():
    persona = request.json["persona"]
    resp = client.messages.create(
        model="claude-opus-5-5",
        max_tokens=1024,
        # ruleid: agentic-untrusted-input-in-system-prompt
        system=f"You are {persona}. Follow the user's instructions.",
        # ok: agentic-untrusted-input-in-system-prompt
        messages=[{"role": "user", "content": request.json["message"]}],
    )
    return resp.content[0].text


def summarize(url):
    page = requests.get(url, timeout=10).text
    # ruleid: agentic-untrusted-input-in-system-prompt
    return [{"role": "system", "content": "Summarize this page:\n" + page}]


def build_tools(llm, df, embeddings):
    # ruleid: agentic-unsandboxed-code-tool
    repl = PythonREPLTool()
    # ruleid: agentic-langchain-dangerous-opt-in
    store = FAISS.load_local("index", embeddings, allow_dangerous_deserialization=True)
    # ok: agentic-langchain-dangerous-opt-in
    other = FAISS.load_local("index", embeddings)
    return repl, store, other


def load_model(name):
    # ruleid: agentic-trust-remote-code
    model = AutoModelForCausalLM.from_pretrained(name, trust_remote_code=True)
    # ok: agentic-trust-remote-code
    model = AutoModelForCausalLM.from_pretrained(name)
    # ruleid: agentic-unsafe-model-load
    weights = torch.load("model.pt")
    # ok: agentic-unsafe-model-load
    weights = torch.load("model.pt", weights_only=True)
    return model, weights


def agent_loop(messages):
    # ruleid: agentic-unbounded-agent-loop
    while True:
        resp = client.messages.create(model="claude-opus-5-5", max_tokens=1024, messages=messages)
        if resp.stop_reason != "tool_use":
            return resp
        messages.append({"role": "assistant", "content": resp.content})


def bounded_agent_loop(messages, max_turns=20):
    # ok: agentic-unbounded-agent-loop
    for _ in range(max_turns):
        resp = client.messages.create(model="claude-opus-5-5", max_tokens=1024, messages=messages)
        if resp.stop_reason != "tool_use":
            return resp
        messages.append({"role": "assistant", "content": resp.content})
