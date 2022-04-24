document.getElementById("save-btn").addEventListener("click", (evt) => {
  evt.preventDefault();
  const id = document.getElementById("save-form").getAttribute("data-entry-id");
  if (!id) {
    return;
  }
  // TODO: Save the changes
});
