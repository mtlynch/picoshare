export function disableElement(el) {
  el.setAttribute("disabled", "true");
}

export function enableElement(el) {
  el.removeAttribute("disabled");
}
