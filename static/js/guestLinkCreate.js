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
  //        Make sure number inputs are ints instead of decimals.
  return {
    label: labelInput.value || null,
    expirationTime: expirationSelect.value,
    sizeLimit: sizeUploadLimitInput.valueAsNumber
      ? megabytesToBytes(sizeUploadLimitInput.valueAsNumber)
      : null,
    countLimit: fileUploadLimitInput.valueAsNumber
      ? fileUploadLimitInput.valueAsNumber
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
