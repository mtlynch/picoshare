export function createSnackbar(message) {
  if (document.querySelectorAll("#snackbar-container").length === 0) {
    const snackbarContainer = document.createElement("div");
    snackbarContainer.setAttribute("id", "snackbar-container");
    document.body.appendChild(snackbarContainer);
  }

  const el = document.createElement("div");
  el.classList.add("snackbar");
  el.innerHTML = message;

  document.getElementById("snackbar-container").append(el);
  el.classList.add("show");

  setTimeout(() => {
    el.remove();
  }, 3000);
}
