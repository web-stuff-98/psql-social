import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
} from "../interfaces/GeneralInterfaces";
import { makeRequest } from "./makeRequest";

export const getBlockedUids = (): Promise<string[] | null> =>
  makeRequest("/api/acc/blocked");

export const getFriendsUids = (): Promise<string[] | null> =>
  makeRequest("/api/acc/friends");

export const getConversationUids = (): Promise<string[] | null> =>
  makeRequest("/api/acc/uids");

export const getConversationContent = (
  uid: string
): Promise<{
  friend_requests: IFriendRequest[] | null;
  invitations: IInvitation[] | null;
  direct_messages: IDirectMessage[] | null;
} | null> => makeRequest(`/api/acc/conv/${uid}`);

export const uploadBio = (content: string): Promise<void> =>
  makeRequest("/api/acc/bio", {
    method: "POST",
    data: { content },
  });

export const uploadPfp = (file: File): Promise<void> => {
  const data = new FormData();
  data.append("file", file);
  return makeRequest("/api/acc/pfp", {
    method: "POST",
    data,
  });
};

export const refreshToken = () =>
  makeRequest("/api/acc/refresh", { method: "POST" });
