"use strict";

export async function uploadFile(file) {
  const formData = new FormData();
  formData.append("file", file);
  return fetch("/api/entry", {
    method: "PUT",
    credentials: "include",
    body: formData,
  })
    .then((response) => {
      if (!response.ok) {
        return response.text().then((error) => {
          return Promise.reject(error);
        });
      }
      return response.json();
    })
    .catch((error) => {
      return Promise.reject(error);
    });
}
