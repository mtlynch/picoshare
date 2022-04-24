import { editFile } from "./controllers/files.js";

function readFilename() {
  return document.getElementById("filename").value || null;
}

function readNote() {
  return document.getElementById("note").value || null;
}

document.getElementById("save-btn").addEventListener("click", (evt) => {
  evt.preventDefault();
  const id = document.getElementById("save-form").getAttribute("data-entry-id");
  if (!id) {
    return;
  }
  editFile(id, readFilename(), readNote())
    .then(() => {
      document.location = "/files";
    })
    .catch((error) => {
      console.error(error); // TODO: Better error handling
    });
});
