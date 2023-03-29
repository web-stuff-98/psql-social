export interface IUser {
  ID: string;
  username: string;
  role: "ADMIN" | "USER";
}
export interface IResMsg {
  msg?: string;
  err?: boolean;
  pen?: boolean;
}
