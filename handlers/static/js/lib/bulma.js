export function hideElement(el) {
  el.classList.add("d-none");
}

export function showElement(el) {
  el.classList.remove("d-none");
}

export function toggleShowElement(el) {
  el.classList.toggle("d-none");
}
