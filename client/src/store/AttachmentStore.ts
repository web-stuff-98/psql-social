import { defineStore } from "pinia";
import { IAttachmentMetadata } from "../interfaces/GeneralInterfaces";
import { baseURL, makeRequest } from "../services/makeRequest";

type DisappearedAttachment = {
  id: string;
  disappearedAt: number;
};

type AttachmentStoreState = {
  attachments: IAttachmentMetadata[];
  visibleAttachments: string[];
  disappearedAttachments: DisappearedAttachment[];
};

const useAttachmentStore = defineStore("attachments", {
  state: () =>
    ({
      attachments: [],
      visibleAttachments: [],
      disappearedAttachments: [],
    } as AttachmentStoreState),
  getters: {
    getRoom(state) {
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
    attachmentEnteredView(id: string) {
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
  },
});

export default useAttachmentStore;
