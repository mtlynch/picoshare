import { logOut } from "./controllers/auth.js";

document.addEventListener("DOMContentLoaded", () => {
  const navbarBurgers = Array.prototype.slice.call(
    document.querySelectorAll(".navbar-burger"),
    0
  );

  if (navbarBurgers.length > 0) {
    navbarBurgers.forEach((el) => {
      el.addEventListener("click", () => {
        const target = document.getElementById(el.dataset.target);
        el.classList.toggle("is-active");
        target.classList.toggle("is-active");
      });
    });
  }
});

const logOutEl = document.getElementById("navbar-log-out");
if (logOutEl) {
  logOutEl.addEventListener("click", () => {
    logOut().then(() => {
      document.location = "/";
    });
  });
}
