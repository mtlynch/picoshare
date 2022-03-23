import { deleteFile } from "./controllers/delete.js";

const uploadEl = document.querySelector(".file");
const resultEl = document.getElementById("upload-result");
const errorContainer = document.getElementById("error");

document.querySelectorAll('[pico-purpose="delete"]').forEach((deleteBtn) => {
  deleteBtn.addEventListener("click", (evt) => {
    const id = deleteBtn.getAttribute("pico-entry-id");
    deleteFile(id)
      .then(() => {
        const rowEl = deleteBtn.parentElement.parentElement;
        rowEl.classList.add("deleted-entry");
      })
      .catch((error) => {
        console.error(error);
      });
  });
});

document.querySelectorAll(["[pico-shortlink]"]).forEach((shortLink) => {
  const id = shortLink.getAttribute("pico-entry-id");
  let btn = shortLink.querySelector("button");
  let input = shortLink.querySelector("input");

  input.value = `${document.location.origin}/!${id}`;

  btn.addEventListener("click", () => {
    input.select();
    input.setSelectionRange(0, 99999);
    navigator.clipboard.writeText(input.value);
  });
});

document.querySelectorAll(["[pico-file-size]"]).forEach((fileSize) => {
  fileSize.innerText = formatBytes(fileSize.innerText);
});

function formatBytes(bytes, decimals = 2) {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}
