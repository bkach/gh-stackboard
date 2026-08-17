const state = { stacks: [], view: "mine", query: "", expanded: new Set() };

const list = document.querySelector("#stack-list");
const template = document.querySelector("#stack-template");
const search = document.querySelector("#search");

function stackStatus(stack) {
  const failing = stack.prs.find((pr) => pr.checks === "failing");
  if (failing) return { label: "CI blocked", tone: "red", action: `Fix CI on #${failing.number}` };
  const changes = stack.prs.find((pr) => pr.review === "changes");
  if (changes) return { label: "Changes needed", tone: "amber", action: `Address review on #${changes.number}` };
  const review = stack.prs.find((pr) => pr.review === "waiting");
  if (review) return { label: "Review needed", tone: "purple", action: `Review #${review.number}` };
  const draft = stack.prs.find((pr) => pr.review === "draft");
  if (draft) return { label: "In progress", tone: "purple", action: `Finish draft #${draft.number}` };
  return { label: "Ready to land", tone: "green", action: `Merge #${stack.prs[0].number}` };
}

function reviewSignal(pr) {
  if (pr.review === "approved") return ["✓ Approved", "pass"];
  if (pr.review === "changes") return ["↻ Changes requested", "fail"];
  if (pr.review === "draft") return ["◌ Draft", "draft"];
  return ["◌ Review pending", "wait"];
}

function checksSignal(pr) {
  if (pr.checks === "passing") return ["✓ Checks passing", "pass"];
  if (pr.checks === "failing") return ["× Checks failing", "fail"];
  return ["◌ Checks running", "wait"];
}

function matchesView(stack) {
  if (state.view === "mine") return stack.mine;
  if (state.view === "assigned") return stack.assigned;
  if (state.view === "team") return stack.team;
  return true;
}

function matchesQuery(stack) {
  const searchable = [stack.title, stack.repository, stack.owner, ...stack.prs.flatMap((pr) => [pr.title, pr.branch, String(pr.number)])].join(" ").toLowerCase();
  return searchable.includes(state.query.toLowerCase());
}

function renderPR(pr, index) {
  const row = document.createElement("div");
  row.className = "pr-item";
  const [review, reviewClass] = reviewSignal(pr);
  const [checks, checksClass] = checksSignal(pr);
  const title = pr.url
    ? `<a href="${escapeHTML(pr.url)}" target="_blank" rel="noreferrer">#${pr.number} ${escapeHTML(pr.title)}</a>`
    : `#${pr.number} ${escapeHTML(pr.title)}`;
  row.innerHTML = `
    <span class="step">${index + 1}</span>
    <span class="pr-copy"><strong>${title}</strong><small><code>${escapeHTML(pr.branch)}</code> · ${escapeHTML(pr.updated)}</small></span>
    <span class="signal checks-signal ${checksClass}">${checks}</span>
    <span class="signal review-signal ${reviewClass}">${review}</span>
    <span class="diff"><b>+${pr.additions}</b><i>−${pr.deletions}</i></span>`;
  return row;
}

function render() {
  const visible = state.stacks.filter((stack) => matchesView(stack) && matchesQuery(stack));
  list.replaceChildren();

  if (!visible.length) {
    const empty = document.createElement("div");
    empty.className = "empty";
    empty.textContent = "No stacks match this view.";
    list.append(empty);
  }

  visible.forEach((stack) => {
    const fragment = template.content.cloneNode(true);
    const article = fragment.querySelector(".stack-row");
    const button = fragment.querySelector(".row-summary");
    const status = stackStatus(stack);
    const expanded = state.expanded.has(stack.id);

    article.dataset.id = stack.id;
    fragment.querySelector(".stack-name strong").textContent = stack.title;
    fragment.querySelector(".stack-name small").textContent = stack.repository;
    fragment.querySelector(".owner i").textContent = stack.initials;
    fragment.querySelector(".owner span").textContent = stack.owner;
    fragment.querySelector(".pr-total").textContent = `${stack.prs.length} PR${stack.prs.length === 1 ? "" : "s"}`;
    const pill = fragment.querySelector(".status-pill");
    pill.textContent = status.label;
    pill.classList.add(status.tone);
    fragment.querySelector(".next-action").textContent = status.action;
    button.setAttribute("aria-expanded", String(expanded));
    button.setAttribute("aria-label", `${stack.title}. ${status.label}. ${expanded ? "Collapse" : "Expand"} stack`);

    const prList = fragment.querySelector(".pr-list");
    stack.prs.slice().reverse().forEach((pr, index) => prList.append(renderPR(pr, stack.prs.length - index - 1)));
    button.addEventListener("click", () => {
      if (state.expanded.has(stack.id)) state.expanded.delete(stack.id);
      else state.expanded.add(stack.id);
      render();
      document.querySelector(`[data-id="${stack.id}"] .row-summary`)?.focus();
    });
    list.append(fragment);
  });

  document.querySelector("#result-summary").textContent = `${visible.length} stack${visible.length === 1 ? "" : "s"} · ${visible.reduce((sum, stack) => sum + stack.prs.length, 0)} pull requests`;
}

function updateCounts() {
  document.querySelector("#count-all").textContent = state.stacks.length;
  document.querySelector("#count-mine").textContent = state.stacks.filter((stack) => stack.mine).length;
  document.querySelector("#count-assigned").textContent = state.stacks.filter((stack) => stack.assigned).length;
  document.querySelector("#count-team").textContent = state.stacks.filter((stack) => stack.team).length;
}

async function load() {
  try {
    const response = await fetch("/api/stacks", { cache: "no-store" });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`);
    state.stacks = data.stacks;
    document.querySelector("#source-label").textContent = data.source === "github" ? `${data.repository}${data.syncing ? " · syncing" : ""}` : "Mock data";
    document.querySelector("#connection-summary").textContent = data.source === "github" ? `Signed in as @${data.viewer.login}` : "Local preview · no GitHub connection";
    const viewerAvatar = document.querySelector("#viewer-avatar");
    viewerAvatar.textContent = data.viewer.login.slice(0, 2).toUpperCase();
    viewerAvatar.setAttribute("aria-label", `Account: ${data.viewer.name || data.viewer.login}`);
    document.querySelectorAll("[data-view]").forEach((tab) => { tab.disabled = false; });
    search.disabled = false;
    updateCounts();
    render();
    if (data.syncing) setTimeout(load, 5_000);
  } catch (error) {
    list.innerHTML = `<div class="error">Could not load stacks. ${escapeHTML(error.message)}</div>`;
  }
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"]/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[character]);
}

document.querySelectorAll("[data-view]").forEach((tab) => {
  tab.addEventListener("click", () => {
    state.view = tab.dataset.view;
    document.querySelectorAll("[data-view]").forEach((item) => item.setAttribute("aria-selected", String(item === tab)));
    render();
  });
});

search.addEventListener("input", () => { state.query = search.value.trim(); render(); });
load();
setInterval(() => { if (!document.hidden) load(); }, 60_000);
document.addEventListener("visibilitychange", () => { if (!document.hidden) load(); });
