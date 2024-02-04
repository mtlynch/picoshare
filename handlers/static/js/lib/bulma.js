export function hideElement(el) {
  el.classList.add("is-hidden");
}

export function showElement(el) {
  el.classList.remove("is-hidden");
}

export function toggleShowElement(el) {
  el.classList.toggle("is-hidden");
}
