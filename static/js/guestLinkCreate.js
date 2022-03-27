import { guestLinkNew } from "./controllers/guestLinks.js";

const labelInput = document.getElementById("label");
const expirationSelect = document.getElementById("expiration-select");
const sizeUploadLimitInput = document.getElementById("size-upload-limit");
const fileUploadLimitInput = document.getElementById("file-upload-limit");
const createBtn = document.getElementById("create-btn");

function megabytesToBytes(megabytes) {
  return megabytes * 1024 * 1024;
}

function guestLinkFromInputs() {
  // TODO: Validate the inputs.
  return {
    label: labelInput.value || null,
    expirationTime: expirationSelect.value,
    sizeLimit: sizeUploadLimitInput.value
      ? megabytesToBytes(parseInt(sizeUploadLimitInput.value))
      : null,
    countLimit: fileUploadLimitInput.value
      ? parseInt(fileUploadLimitInput.value)
      : null,
  };
}

// TODO: Probably want a normal form submit so that it works for button click,
// keyboard, etc.
createBtn.addEventListener("click", () => {
  guestLinkNew(guestLinkFromInputs()).then(() => {
    document.location = "/guest-links";
  });
});
