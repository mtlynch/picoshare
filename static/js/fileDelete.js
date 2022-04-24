import { deleteFile } from "./controllers/files.js";

document.getElementById("delete-btn").addEventListener("click", (evt) => {
  evt.preventDefault();
  const id = evt.target.getAttribute("data-entry-id");
  if (!id) {
    return;
  }
  deleteFile(id)
    .then(() => {
      document.location = "/files";
    })
    .catch((error) => {
      console.error(error); // TODO: Better error handling
    });
});
