import Anthropic from "@anthropic-ai/sdk";
import OpenAI from "openai";
import { exec } from "child_process";
import express from "express";

const anthropic = new Anthropic();
const app = express();

async function runGenerated(task) {
  const resp = await anthropic.messages.create({
    model: "claude-opus-5-5",
    max_tokens: 1024,
    messages: [{ role: "user", content: task }],
  });
  const code = resp.content[0].text;
  // ruleid: agentic-js-llm-output-to-code-execution
  eval(code);
}

async function runCommand(task) {
  const openai = new OpenAI();
  const completion = await openai.chat.completions.create({
    model: "gpt-5",
    messages: [{ role: "user", content: task }],
  });
  // ruleid: agentic-js-llm-output-to-code-execution
  exec(completion.choices[0].message.content);
  // ok: agentic-js-llm-output-to-code-execution
  exec("git status");
}

async function renderAnswer(el, question) {
  const resp = await anthropic.messages.create({
    model: "claude-opus-5-5",
    max_tokens: 1024,
    messages: [{ role: "user", content: question }],
  });
  // ruleid: agentic-js-llm-output-to-html
  el.innerHTML = marked.parse(resp.content[0].text);
  // ok: agentic-js-llm-output-to-html
  el.innerHTML = DOMPurify.sanitize(marked.parse(resp.content[0].text));
  // ok: agentic-js-llm-output-to-html
  el.textContent = resp.content[0].text;
}

app.post("/chat", async (req, res) => {
  const resp = await anthropic.messages.create({
    model: "claude-opus-5-5",
    max_tokens: 1024,
    // ruleid: agentic-js-untrusted-input-in-system-prompt
    system: `You are a support assistant for ${req.body.company}.`,
    // ok: agentic-js-untrusted-input-in-system-prompt
    messages: [{ role: "user", content: req.body.message }],
  });
  res.json(resp);
});

// ruleid: agentic-js-browser-api-key-exposure
const browserClient = new Anthropic({ dangerouslyAllowBrowser: true });
// ok: agentic-js-browser-api-key-exposure
const serverClient = new Anthropic({ maxRetries: 3 });
