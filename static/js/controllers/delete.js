"use strict";

export async function deleteFile(id) {
  return fetch(`/api/entry/${id}`, {
    method: "DELETE",
    credentials: "include",
  })
    .then((response) => {
      if (!response.ok) {
        return response.text().then((error) => {
          return Promise.reject(error);
        });
      }
      return Promise.resolve();
    })
    .catch((error) => {
      return Promise.reject(error);
    });
}
