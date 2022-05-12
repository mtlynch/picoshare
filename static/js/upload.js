import { guestUploadFile, uploadFile } from "./controllers/files.js";
import { showElement, hideElement } from "./lib/bulma.js";

const uploadEl = document.querySelector(".file");
const resultEl = document.getElementById("upload-result");
const pasteEl = document.getElementById("pastebox");
const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const uploadForm = document.getElementById("upload-form");
const expirationContainer = document.querySelector(".expiration-container");
const expirationSelect = document.getElementById("expiration-select");
const noteInput = document.getElementById("note");
const uploadAnotherBtn = document.getElementById("upload-another-btn");

function getGuestLinkMetdata() {
  const el = document.getElementById("guest-link-metadata");
  if (!el) {
    return null;
  }
  return JSON.parse(el.innerHTML);
}

function readNote() {
  return noteInput.value || null;
}

function populateEditButton(entryId) {
  const btn = document.getElementById("edit-btn");
  // Button does not appear in guest mode.
  if (!btn) {
    return;
  }

  btn.href = `/files/${entryId}/edit`;
}

// Sorts clipboard items in descending priority of which format we think the
// user wants to upload.
function sortClipboardItems(items) {
  return items.sort((a, b) => {
    if (a.kind === "string" && b.kind === "string") {
      // Prefer text/plain strings ahead of other string types.
      return a.type === "text/plain" ? -1 : 1;
    }
    return 0;
  });
}

function doUpload(file) {
  const guestLinkMetadata = getGuestLinkMetdata();

  if (
    guestLinkMetadata &&
    guestLinkMetadata.maxFileBytes &&
    file.size > guestLinkMetadata.maxFileBytes
  ) {
    const friendlySize = `${guestLinkMetadata.maxFileBytes} bytes`;
    document.getElementById(
      "error-message"
    ).innerText = `File is too large. Maximum upload size is ${friendlySize}.`;
    showElement(errorContainer);
    return;
  }
  hideElement(errorContainer);
  hideElement(uploadForm);
  showElement(progressSpinner);

  let uploader = () => {
    return uploadFile(file, expirationSelect.value, readNote());
  };
  if (guestLinkMetadata) {
    uploader = () => {
      return guestUploadFile(file, guestLinkMetadata.id);
    };
  }
  uploader()
    .then((res) => {
      const entryId = res.id;

      populateEditButton(entryId);

      const uploadLinksEl = document.createElement("upload-links");
      uploadLinksEl.fileId = entryId;
      uploadLinksEl.filename = file.name;
      uploadLinksEl.addEventListener("link-copied", () => {
        document
          .querySelector("snackbar-notifications")
          .addInfoMessage("Copied link");
      });
      document.getElementById("result-links").append(uploadLinksEl);
      showElement(resultEl);
      showElement(uploadAnotherBtn);

      uploadEl.style.display = "none";
      if (expirationContainer) {
        expirationContainer.style.display = "none";
      }
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
  for (const item of sortClipboardItems(Array.from(evt.clipboardData.items))) {
    if (item.kind === "string") {
      item.getAsString((s) => {
        doUpload(
          new File([new Blob([s])], `pasted-${timestamp}.txt`, {
            type: "text/plain",
          })
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

uploadAnotherBtn.addEventListener("click", () => {
  window.location.reload();
});

document.addEventListener("DOMContentLoaded", function () {
  resetPasteInstructions();
  // Set initial focus to paste element so that if the user pastes on page load,
  // it pastes to the right place without them having to manually place the
  // cursor in the right place.
  pasteEl.focus();
});
