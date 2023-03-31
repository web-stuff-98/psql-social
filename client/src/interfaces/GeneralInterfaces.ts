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
export interface IRoom {
  ID: string;
  name: string;
  author_id: string;
  is_private: boolean;
  created_at: string;
}
export interface IRoomChannel {
  ID: string;
  name: string;
  main: boolean;
}
export interface IRoomMessage {
  ID: string;
  content: string;
  created_at: string;
  author_id: string;
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
