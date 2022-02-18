import { uploadFile } from "./controllers/upload.js";

const uploadEl = document.querySelector(".file");
const resultEl = document.getElementById("upload-result");

document
  .querySelector('.file-input[name="resume"]')
  .addEventListener("change", (evt) => {
    resultEl.innerText = "";
    uploadFile(evt.target.files[0])
      .then((res) => {
        console.log(res);

        const aEl = document.createElement("a");

        aEl.href = "/!" + res.ID;
        aEl.innerText = document.location.href + "!" + res.ID;

        resultEl.appendChild(aEl);
        uploadEl.style.display = "none";
      })
      .catch((error) => {
        resultEl.innerText = error;
      });
  });
