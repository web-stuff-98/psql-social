export function validateUsername(username: string) {
  if (!username) return "Username is required";
  if (username.length > 16) return "Maximum 16 characters";
  if (username.length < 2) return "Minimum 2 characters";
  return true;
}

export function validatePassword(password: string) {
  if (!password) return "Password is required";
  if (password.length > 72) return "Maximum 72 characters";
  if (password.length < 8) return "Minimum 8 characters";

  let count = 0;
  if (password.match(/.*\d.*/)) count++;
  if (password.match(/.*[a-z].*/)) count++;
  if (password.match(/.*[A-Z].*/)) count++;
  if (password.match(/.*[*.!@#$%^&(){}[\]:;<>,.?/~`_+-=|\\].*/)) count++;

  return count >= 3
    ? true
    : "Password must contain a number, lowercase letter, uppercase letter and a special character";
}

export const validateBio = (bio: string) =>
  bio.length > 300 ? "Maximum 300 characters" : true;

export const validateRoomName = (name: string) => {
  if (!name) return "Name is required";
  if (name.length < 2) return "Minimum 2 characters";
  if (name.length > 16) return "Maximum 16 characters";
  return true;
};
