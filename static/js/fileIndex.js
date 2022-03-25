import { deleteFile } from "./controllers/delete.js";

const errorContainer = document.getElementById("error");

function hideElement(el) {
  el.classList.add("is-hidden");
}

function showElement(el) {
  el.classList.remove("is-hidden");
}

document.querySelectorAll('[pico-purpose="delete"]').forEach((deleteBtn) => {
  deleteBtn.addEventListener("click", () => {
    const id = deleteBtn.getAttribute("pico-entry-id");
    deleteFile(id)
      .then(() => {
        const rowEl = deleteBtn.parentElement.parentElement;
        rowEl.classList.add("deleted-entry");
      })
      .catch((error) => {
        document.getElementById("error-message").innerText = error;
        showElement(errorContainer);
      });
  });
});

document.querySelector("#error .delete").addEventListener("click", () => {
  hideElement(errorContainer);
});

document.querySelectorAll('[pico-purpose="copy"]').forEach((copyBtn) => {
  copyBtn.addEventListener("click", () => {
    const picoId = copyBtn.getAttribute("pico-entry-id");
    const shortLink = `${window.location.origin}/!${picoId}`;

    copyToClipboard(shortLink)
      .then(() => console.log("text copied !"))
      .catch(() => console.log("error"));
  });
});

/**
 * @param {string} textToCopy
 * @returns Promise
 */
function copyToClipboard(textToCopy) {
  // navigator clipboard api needs a secure context (https)
  if (navigator.clipboard && window.isSecureContext) {
    // navigator clipboard api method'
    return navigator.clipboard.writeText(textToCopy);
  } else {
    // text area method
    const textArea = document.createElement("textarea");
    textArea.value = textToCopy;
    // make the textarea out of viewport
    textArea.style.position = "fixed";
    textArea.style.left = "-999999px";
    textArea.style.top = "-999999px";
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    return new Promise((res, rej) => {
      // here the magic happens
      document.execCommand("copy") ? res() : rej();
      textArea.remove();
    });
  }
}
