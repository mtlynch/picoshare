"use strict";

export async function uploadFile(file, expirationTime) {
  const formData = new FormData();
  formData.append("file", file);
  return fetch(`/api/entry?expiration=${expirationTime}`, {
    method: "POST",
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
    .then((data) => {
      if (!data.hasOwnProperty("id")) {
        throw new Error("Missing expected id field");
      }
      return Promise.resolve(data);
    })
    .catch((error) => {
      return Promise.reject(error);
    });
}
