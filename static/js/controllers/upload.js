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
      console.log(response.ok);
      if (!response.ok) {
        console.log("handling not-ok request");
        return response.text().then((error) => {
          return Promise.reject(error);
        });
      }
      console.log("handling json");
      return response.json();
    })
    .catch((error) => {
      return Promise.reject(error);
    });
}
