import { defineStore } from "pinia";
import { IAttachmentMetadata } from "../interfaces/GeneralInterfaces";
import { baseURL, makeRequest } from "../services/makeRequest";
import {
  isAttachmentMetadataCreated,
  isAttachmentProgress,
} from "../socketHandling/InterpretEvent";

type DisappearedAttachment = {
  id: string;
  disappearedAt: number;
};

type thisState = {
  attachments: IAttachmentMetadata[];
  visibleAttachments: string[];
  disappearedAttachments: DisappearedAttachment[];
};

const usethis = defineStore("attachments", {
  state: () =>
    ({
      attachments: [],
      visibleAttachments: [],
      disappearedAttachments: [],
    } as thisState),
    
  getters: {
    getAttachment(state) {
      return (id: string) => state.attachments.find((a) => a.ID === id);
    },
  },

  actions: {
    async cacheAttachment(id: string, force?: boolean) {
      if (
        this.$state.attachments.findIndex((a) => a.ID === id) !== -1 &&
        !force
      )
        return;
      try {
        const a: IAttachmentMetadata = await makeRequest(
          `${baseURL}/api/attachment/${id}`
        );
        // spread operator to make sure DOM updates, not sure if necessary
        this.$state.attachments = [
          ...this.$state.attachments.filter((a) => a.ID !== id),
          a,
        ];
      } catch (e) {
        console.warn("Failed to cache attachment data for", id);
      }
    },

    cleanupInterval() {
      const disappeared = this.disappearedAttachments.map((da) =>
        Date.now() - da.disappearedAt > 30000 ? da.id : ""
      );
      this.attachments = [
        ...this.attachments.filter((a) => !disappeared.includes(a.ID)),
      ];
      this.disappearedAttachments = [
        ...this.disappearedAttachments.filter(
          (da) => !disappeared.includes(da.id)
        ),
      ];
    },

    watchAttachments(e: MessageEvent) {
      const msg = JSON.parse(e.data);
      if (!msg) return;
      if (isAttachmentProgress(msg)) {
        const i = this.attachments.findIndex((a) => a.ID === msg.data.ID);
        if (i !== -1) {
          const newMeta = {
            ...this.attachments[i],
            ratio: msg.data.ratio,
            failed: msg.data.failed,
          };
          this.attachments = [
            ...this.attachments.filter((a) => a.ID !== msg.data.ID),
            newMeta,
          ];
        }
      }
      if (isAttachmentMetadataCreated(msg)) {
        this.attachments = [
          ...this.attachments.filter((a) => a.ID !== msg.data.ID),
          {
            ...msg.data,
            failed: false,
            ratio: 0,
          },
        ];
      }
    },

    attachmentEnteredView(id: string) {
      this.cacheAttachment(id);
      this.$state.visibleAttachments = [...this.$state.visibleAttachments, id];
      const i = this.$state.disappearedAttachments.findIndex(
        (a) => a.id === id
      );
      if (i !== -1) this.$state.disappearedAttachments.splice(i, 1);
    },
    attachmentLeftView(id: string) {
      const i = this.$state.visibleAttachments.findIndex((a) => a === id);
      if (i !== -1) this.$state.visibleAttachments.splice(i, 1);
      if (
        this.$state.disappearedAttachments.findIndex((a) => a.id === id) === -1
      )
        this.$state.disappearedAttachments = [
          ...this.$state.disappearedAttachments,
          {
            id,
            disappearedAt: Date.now(),
          },
        ];
    },
    async uploadAttachment(file: File, id: string) {
      await makeRequest(`${baseURL}/api/attachment/metadata`, {
        data: {
          name: file.name,
          size: file.size,
          mime: file.type,
          msg_id: id,
        },
        method: "POST",
      });
      // Split attachment into 4mb chunks
      let fileUploadChunks: Promise<ArrayBuffer>[] = [];
      let startPointer = 0;
      let endPointer = file.size;
      while (startPointer < endPointer) {
        let newStartPointer = startPointer + 4 * 1024 * 1024;
        fileUploadChunks.push(
          new Blob([file.slice(startPointer, newStartPointer)]).arrayBuffer()
        );
        startPointer = newStartPointer;
      }
      // Upload chunks
      for await (const data of fileUploadChunks) {
        await makeRequest(`${baseURL}/api/attachment/chunk/${id}`, {
          method: "POST",
          headers: { "Content-Type": "application/octet-stream" },
          data,
        });
      }
    },
  },
});

export default usethis;
