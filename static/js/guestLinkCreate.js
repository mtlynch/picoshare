import { guestLinkNew } from "./controllers/guestLinks.js";

const labelInput = document.getElementById("label");
const expirationSelect = document.getElementById("expiration-select");
const maxFileBytesInput = document.getElementById("max-file-size");
const fileUploadLimitInput = document.getElementById("file-upload-limit");
const createLinkForm = document.getElementById("create-guest-link-form");

function megabytesToBytes(megabytes) {
  return megabytes * 1024 * 1024;
}

function guestLinkFromInputs() {
  return {
    label: labelInput.value || null,
    expirationTime: expirationSelect.value,
    maxFileBytes: maxFileBytesInput.valueAsNumber
      ? megabytesToBytes(maxFileBytesInput.valueAsNumber)
      : null,
    maxFileUploads: fileUploadLimitInput.valueAsNumber
      ? fileUploadLimitInput.valueAsNumber
      : null,
  };
}

createLinkForm.addEventListener("submit", (evt) => {
  evt.preventDefault();
  const guestLink = guestLinkFromInputs();
  guestLinkNew(
    guestLink.label,
    guestLink.expirationTime,
    guestLink.maxFileBytes,
    guestLink.maxFileUploads
  ).then(() => {
    document.location = "/guest-links";
  });
});
