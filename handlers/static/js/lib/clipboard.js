export function copyToClipboard(textToCopy) {
  if (navigator.clipboard && window.isSecureContext) {
    return writeToClipboard(textToCopy);
  } else {
    return legacyWriteToClipboard(textToCopy);
  }
}

// Sorts clipboard items in descending priority of which format we think the
// user wants to upload.
export function sortClipboardItems(items) {
  return items.sort((a, b) => {
    // Prioritize images ahead of other formats.
    const isImage = (x) => {
      return x.type.startsWith("image");
    };
    if (isImage(a) && !isImage(b)) {
      return -1;
    }
    if (isImage(b) && !isImage(a)) {
      return 1;
    }

    if (a.kind === "string" && b.kind === "string") {
      // Prefer text/plain strings ahead of other string types.
      return a.type === "text/plain" ? -1 : 1;
    }
    return 0;
  });
}

// Copy text to clipboard using the modern clipboard API.
function writeToClipboard(textToWrite) {
  return navigator.clipboard.writeText(textToWrite);
}

// Copy text to clipboard using legacy APIs that work even if serving over a
// non-HTTPS connection.
function legacyWriteToClipboard(textToWriter) {
  const textArea = document.createElement("textarea");
  textArea.value = textToWriter;

  // Position the textarea off-screen.
  textArea.style.position = "fixed";
  textArea.style.left = "-999999px";
  textArea.style.top = "-999999px";
  document.body.appendChild(textArea);

  textArea.focus();
  textArea.select();
  return new Promise((resolve, reject) => {
    document.execCommand("copy") ? resolve() : reject();
    textArea.remove();
  });
}
