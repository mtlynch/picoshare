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

function doUpload(file) {
  hideElement(errorContainer);
  hideElement(uploadForm);
  showElement(progressSpinner);
  uploadFile(file, expirationSelect.value)
    .then((res) => {
      const entryId = res.id;

      const uploadLinksEl = document.createElement("upload-links");
      uploadLinksEl.fileId = entryId;
      uploadLinksEl.filename = file.name;
      uploadLinksEl.addEventListener("link-copied", () => {
        document
          .querySelector("snackbar-notifications")
          .addInfoMessage("Copied link");
      });
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
    doUpload(evt.target.files[0]);
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
      doUpload(file);
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
  const timestamp = new Date().toISOString().replaceAll(":", "");
  for (const item of evt.clipboardData.items) {
    if (item.kind === "string") {
      item.getAsString((s) => {
        doUpload(
          new File([new Blob([s])], `pasted-${timestamp}.txt`, {
            type: "text/plain",
          }),
          expirationSelect.value
        );
      });
      return;
    }
    let pastedFile = item.getAsFile();
    if (!pastedFile) {
      continue;
    }

    // Pasted images are named image.png by default, so make a better filename.
    if (pastedFile.name === "image.png") {
      pastedFile = new File([pastedFile], `pasted-${timestamp}.png`, {
        type: pastedFile.type,
      });
    }

    doUpload(pastedFile);
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
