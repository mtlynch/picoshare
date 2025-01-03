"use strict";

export async function guestLinkNew(
  label,
  urlExpirationTime,
  fileLifetime,
  maxFileBytes,
  maxFileUploads
) {
  return fetch("/api/guest-links", {
    method: "POST",
    credentials: "include",
    body: JSON.stringify({
      label,
      urlExpirationTime,
      fileLifetime,
      maxFileBytes,
      maxFileUploads,
    }),
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
          "Failed to communicate with server" +
            (error.message ? `: ${error.message}` : ".")
        );
      }
      return Promise.reject(error);
    });
}

export async function guestLinkDelete(id) {
  return fetch(`/api/guest-links/${id}`, {
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
          "Failed to communicate with server" +
            (error.message ? `: ${error.message}` : ".")
        );
      }
      return Promise.reject(error);
    });
}

export async function guestLinkEnable(id) {
  return fetch(`/api/guest-links/enable/${id}`, {
    method: "PATCH",
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
          "Failed to communicate with server" +
            (error.message ? `: ${error.message}` : ".")
        );
      }
      return Promise.reject(error);
    });
}

export async function guestLinkDisable(id) {
  return fetch(`/api/guest-links/disable/${id}`, {
    method: "PATCH",
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
          "Failed to communicate with server" +
            (error.message ? `: ${error.message}` : ".")
        );
      }
      return Promise.reject(error);
    });
}
