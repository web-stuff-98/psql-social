export interface IUser {
  ID: string;
  username: string;
  role: "ADMIN" | "USER";
  // pfp is an object url this time. not base64
  pfp?: string;
}
export interface IResMsg {
  msg?: string;
  err?: boolean;
  pen?: boolean;
}
export interface IMediaOptions {
  userMedia: {
    video: boolean;
    audio: boolean;
  };
  displayMedia: {
    video: boolean;
  };
}
