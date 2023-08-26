"use strict";

export async function settingsPut(settings) {
  return fetch("/api/settings", {
    method: "PUT",
    credentials: "include",
    body: JSON.stringify(settings),
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
