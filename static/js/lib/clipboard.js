export function copyToClipboard(textToCopy) {
  if (navigator.clipboard && window.isSecureContext) {
    return writeToClipboard(textToCopy);
  } else {
    return legacyWriteToClipboard(textToCopy);
  }
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
