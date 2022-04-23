"use strict";

export async function uploadFile(file, expirationTime, note) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("note", note);
  return fetch(`/api/entry?expiration=${encodeURIComponent(expirationTime)}`, {
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
      if (!Object.prototype.hasOwnProperty.call(data, "id")) {
        throw new Error("Missing expected id field");
      }
      return Promise.resolve(data);
    })
    .catch((error) => {
      if (error.message) {
        return Promise.reject(
          "Failed to communicate with server: " + error.message
        );
      }
      return Promise.reject(error);
    });
}

export async function guestUploadFile(file, guestLinkID) {
  const formData = new FormData();
  formData.append("file", file);
  return fetch(`/api/guest/${guestLinkID}`, {
    method: "POST",
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
      if (!Object.prototype.hasOwnProperty.call(data, "id")) {
        throw new Error("Missing expected id field");
      }
      return Promise.resolve(data);
    })
    .catch((error) => {
      if (error.message) {
        return Promise.reject(
          "Failed to communicate with server: " + error.message
        );
      }
      return Promise.reject(error);
    });
}
