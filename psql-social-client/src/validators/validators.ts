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
  return true;
}
