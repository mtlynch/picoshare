const labelInput = document.getElementById("label");
const expirationSelect = document.getElementById("expiration-select");
const sizeUploadLimitInput = document.getElementById("size-upload-limit");
const fileUploadLimitInput = document.getElementById("file-upload-limit");
const createBtn = document.getElementById("create-btn");

function payloadFromForm() {
  return {
    label: labelInput.value,
    expirationTime: expirationSelect.value,
    sizeUploadLimit: sizeUploadLimitInput.value,
    fileUploadLimit: fileUploadLimitInput.value,
  };
}

// TODO: Probably want a normal form submit so that it works for button click,
// keyboard, etc.
createBtn.addEventListener("click", () => {
  //console.log("clicked create");
  //console.log(payloadFromForm());
  payloadFromForm();
});
