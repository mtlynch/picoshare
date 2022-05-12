"use strict";

export async function uploadFile(file, expirationTime, note) {
  const formData = new FormData();
  formData.append("file", file);
  if (note) {
    formData.append("note", note);
  }
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

export async function editFile(id, filename, expiration, note) {
  let payload = {
    filename,
    note,
  };
  if (expiration) {
    payload.expiration = expiration;
  }
  return fetch(`/api/entry/${encodeURIComponent(id)}`, {
    method: "PUT",
    credentials: "include",
    body: JSON.stringify(payload),
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
      if (error.message) {
        return Promise.reject(
          "Failed to communicate with server: " + error.message
        );
      }
      return Promise.reject(error);
    });
}

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
      if (error.message) {
        return Promise.reject(
          "Failed to communicate with server: " + error.message
        );
      }
      return Promise.reject(error);
    });
}
