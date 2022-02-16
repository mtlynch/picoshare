export async function authenticate(passphrase) {
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

export function logOut() {
  document.cookie = "sharedSecret=; Max-Age=-99999999;";
}
