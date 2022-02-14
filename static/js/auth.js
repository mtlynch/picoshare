async function authenticate(passphrase) {
  return fetch("/api/auth", {
    method: "POST",
    mode: "same-origin",
    credentials: "include",
    cache: "no-cache",
    redirect: "error",
    body: JSON.stringify({
      sharedSecret: passphrase,
    }),
  }).then((response) => {
    if (!response.ok) {
      return response.text().then((error) => {
        return Promise.reject(error);
      });
    }
    return Promise.resolve();
  });
}

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
      document.location = "/admin/";
    })
    .catch((error) => {
      document.cookie = "sharedSecret=; Max-Age=-99999999;";
      document.getElementById("error-message").innerText = error;
      errorContainer.classList.remove("is-hidden");
      enableAuthForm();
    });
});
