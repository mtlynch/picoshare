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
  const btn = shortLink.querySelector("button");

  btn.addEventListener("click", () => {
    const input = shortLink.querySelector("input");

    input.select();
    input.setSelectionRange(0, 99999);

    if (navigator && navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard
        .writeText(input.value)
        .then(() => {
          console.log("Text copied");
        })
        .catch((err) => {
          console.log("Something went wrong", err);
        });
    }
  });
});
