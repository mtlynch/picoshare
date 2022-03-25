import { uploadFile } from "./controllers/upload.js";

const uploadEl = document.querySelector(".file");
const resultEl = document.getElementById("upload-result");
const pasteEl = document.getElementById("pastebox");
const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const uploadForm = document.getElementById("upload-form");
const expirationContainer = document.querySelector(".expiration-container");
const expirationSelect = document.getElementById("expiration-select");

function hideElement(el) {
  el.classList.add("is-hidden");
}

function showElement(el) {
  el.classList.remove("is-hidden");
}

function doUpload(file, expiration) {
  hideElement(errorContainer);
  hideElement(uploadForm);
  showElement(progressSpinner);
  uploadFile(file, expiration)
    .then((res) => {
      const entryId = res.id;

      const uploadLinksEl = document.createElement("upload-links");
      uploadLinksEl.fileId = entryId;
      uploadLinksEl.filename = file.name;
      resultEl.append(uploadLinksEl);
      showElement(resultEl);

      uploadEl.style.display = "none";
      expirationContainer.style.display = "none";
    })
    .catch((error) => {
      document.getElementById("error-message").innerText = error;
      showElement(errorContainer);
      showElement(uploadForm);
    })
    .finally(() => {
      hideElement(progressSpinner);
    });
}

function resetPasteInstructions() {
  pasteEl.value = "Or paste something here";
}

document
  .querySelector('.file-input[name="resume"]')
  .addEventListener("change", (evt) => {
    doUpload(evt.target.files[0], expirationSelect.value);
  });

uploadForm.addEventListener("drop", (evt) => {
  evt.preventDefault();

  uploadForm.classList.remove("accepting-drop");

  if (!evt.dataTransfer.items) {
    return;
  }
  for (var i = 0; i < evt.dataTransfer.items.length; i++) {
    if (evt.dataTransfer.items[i].kind === "file") {
      var file = evt.dataTransfer.items[i].getAsFile();
      doUpload(file, expirationSelect.value);
      return;
    }
  }
});

uploadEl.addEventListener("dragover", (evt) => {
  evt.preventDefault();

  uploadEl.classList.add("accepting-drop");
});

uploadEl.addEventListener("dragenter", (evt) => {
  evt.preventDefault();

  uploadEl.classList.add("accepting-drop");
});

uploadEl.addEventListener("dragleave", () => {
  uploadEl.classList.remove("accepting-drop");
});

pasteEl.addEventListener("paste", (evt) => {
  for (const item of evt.clipboardData.items) {
    if (item.kind === "string") {
      item.getAsString((s) => {
        const timestamp = new Date().toISOString().replaceAll(":", "");
        doUpload(
          new File([new Blob([s])], `pasted-${timestamp}.txt`),
          expirationSelect.value
        );
      });
      return;
    }
    const pastedFile = item.getAsFile();
    if (!pastedFile) {
      continue;
    }

    doUpload(pastedFile, expirationSelect.value);
    return;
  }
});

pasteEl.addEventListener("change", (evt) => {
  evt.preventDefault();
  resetPasteInstructions();
});

pasteEl.addEventListener("input", (evt) => {
  evt.preventDefault();
  resetPasteInstructions();
});

document.addEventListener("DOMContentLoaded", function () {
  resetPasteInstructions();
  // Set initial focus to paste element so that if the user pastes on page load,
  // it pastes to the right place without them having to manually place the
  // cursor in the right place.
  pasteEl.focus();
});
