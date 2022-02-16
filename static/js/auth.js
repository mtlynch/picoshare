import { authenticate, logOut } from "./controllers/auth.js";

function setAuthFormState(isEnabled) {
  document.querySelectorAll("#auth-form input").forEach((el) => {
    el.disabled = !isEnabled;
  });
}

function disableAuthForm() {
  setAuthFormState(/* isEnabled= */ false);
}

function enableAuthForm() {
  setAuthFormState(/* isEnabled= */ true);
}

const errorContainer = document.getElementById("error");
const authForm = document.getElementById("auth-form");
authForm.addEventListener("submit", (evt) => {
  evt.preventDefault();
  const secret = document.getElementById("secret").value;
  errorContainer.classList.add("is-hidden");
  disableAuthForm();
  authenticate(secret)
    .then(() => {
      document.location = "/";
    })
    .catch((error) => {
      logOut();
      document.getElementById("error-message").innerText = error;
      errorContainer.classList.remove("is-hidden");
      enableAuthForm();
    });
});
