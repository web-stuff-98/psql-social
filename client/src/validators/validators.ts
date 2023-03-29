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

  const regex =
    /^(?=.*[A-Za-z])(?=.*\d)(?=.*[@$!%*#?&])[A-Za-z\d@$!%*#?&]{8,}$/;
  if (!regex.test(password))
    return "Password must contain at least one uppercase letter, one lowercase letter and a number";

  //Minimum eight characters, at least one uppercase letter, one lowercase letter and one number
  return true;
}
