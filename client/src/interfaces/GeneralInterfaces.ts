export interface IUser {
  ID: string;
  username: string;
  role: "ADMIN" | "USER";
  online?: boolean;
  // pfp is an object url this time. not base64
  pfp?: string;
}
export interface IResMsg {
  msg?: string;
  err?: boolean;
  pen?: boolean;
}
export interface IAttachmentMetadata {
  ID: string;
  mime: string;
  name: string;
  size: number;
  ratio: number;
  failed: boolean;
}
export interface IRoom {
  ID: string;
  name: string;
  author_id: string;
  is_private: boolean;
  created_at: string;
  // img (object url)
  img?: string;
}
export interface IRoomChannel {
  ID: string;
  name: string;
  main: boolean;
}
export interface IMessage {
  ID: string;
  content: string;
  created_at: string;
  author_id: string;
  has_attachment?: boolean;
}
export interface IRoomMessage extends IMessage {}
export interface IDirectMessage extends IMessage {}
export interface IInvitation {
  inviter: string;
  invited: string;
  room_id: string;
  created_at: string;
  accepted?: boolean;
}
export interface IFriendRequest {
  friender: string;
  friended: string;
  created_at: string;
  accepted?: boolean;
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
