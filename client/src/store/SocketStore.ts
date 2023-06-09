import { defineStore } from "pinia";

type SocketStoreState = {
  socket?: WebSocket;
  currentlyWatching: string[];
};

const useSocketStore = defineStore("socket", {
  state: () =>
    ({
      socket: undefined,
      // currentlyWatching keeps IDs of things that the client is already "watching" so that the
      // client doesn't resend START_WATCHING events for things it's already watching
      currentlyWatching: [],
    } as SocketStoreState),
  actions: {
    send(data: "PING" | object) {
      if (this.socket?.readyState === 1)
        this.socket.send(
          typeof data === "object" ? JSON.stringify(data) : data
        );
      else console.warn("Socket unavailable");
    },

    async connectSocket() {
      return new Promise<WebSocket>((resolve, reject) => {
        const socket = new WebSocket(
          process.env.NODE_ENV === "development" ||
          window.location.origin === "http://localhost:8080/"
            ? `ws://localhost:8080/api/ws`
            : `wss://psql-social.herokuapp.com/api/ws`
        );
        socket.onopen = () => {
          this.socket = socket;
          resolve(socket);
        };
        socket.onerror = (e) => reject(e);
        socket.onclose = () => (this.socket = undefined);
      });
    },
  },
});

export default useSocketStore;
