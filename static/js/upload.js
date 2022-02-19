import { uploadFile } from "./controllers/upload.js";

const uploadEl = document.querySelector(".file");
const resultEl = document.getElementById("upload-result");
const errorContainer = document.getElementById("error");

document
  .querySelector('.file-input[name="resume"]')
  .addEventListener("change", (evt) => {
    errorContainer.classList.add("is-hidden");
    uploadFile(evt.target.files[0])
      .then((res) => {
        const entryId = res.id;

        const aEl = document.createElement("a");

        aEl.href = `/!${entryId}`;
        aEl.innerText = `${document.location.href}!${entryId}`;

        resultEl.appendChild(aEl);
        uploadEl.style.display = "none";
      })
      .catch((error) => {
        document.getElementById("error-message").innerText = error;
        errorContainer.classList.remove("is-hidden");
      });
  });
