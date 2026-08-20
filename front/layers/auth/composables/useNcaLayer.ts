// Обёртка над ncalayer-js-client: подключение к NCALayer (wss://127.0.0.1:13579)
// и создание CMS-подписи над challenge. Работает только в браузере.

interface NcaClientInstance {
  connect(): Promise<void>
  basicsSignCMS(storage: string, dataBase64: string, params: string, signer: string): Promise<string>
}

interface NcaCtor {
  new (): NcaClientInstance
  basicsStorageAll: string
  basicsCMSParamsDetached: string
  basicsSignerSignAny: string
}

export function useNcaLayer() {
  const signChallenge = async (challengeBase64: string): Promise<string> => {
    if (!import.meta.client) {
      throw new Error('NCALayer доступен только в браузере')
    }
    // Динамический импорт: библиотека обращается к WebSocket и не должна попадать в SSR-бандл.
    // ncalayer-js-client (CJS) экспортирует именованный класс NCALayerClient.
    const mod = (await import('ncalayer-js-client')) as unknown as {
      NCALayerClient?: NcaCtor
      default?: { NCALayerClient?: NcaCtor } & NcaCtor
    }
    const NCALayerClient: NcaCtor | undefined =
      mod.NCALayerClient ?? mod.default?.NCALayerClient ?? mod.default
    if (typeof NCALayerClient !== 'function') {
      throw new Error('ncalayer-js-client: не найден конструктор NCALayerClient')
    }
    const client = new NCALayerClient()

    await client.connect()

    // Отделённая (detached) CMS над документом; авто-выбор хранилища и любого подписанта.
    const cms: string = await client.basicsSignCMS(
      NCALayerClient.basicsStorageAll,
      challengeBase64,
      NCALayerClient.basicsCMSParamsDetached,
      NCALayerClient.basicsSignerSignAny,
    )
    return cms
  }

  return { signChallenge }
}

// ncaErrorMessage — понятное сообщение для типовых сбоев NCALayer.
export function ncaErrorMessage(e: unknown): string {
  const msg = (e as { message?: string })?.message || String(e)
  if (/websocket|connect|ECONNREFUSED|1006|closed/i.test(msg)) {
    return 'Не удалось подключиться к NCALayer. Убедитесь, что приложение установлено и запущено.'
  }
  if (/cancel|отмен|reject/i.test(msg)) {
    return 'Подписание отменено.'
  }
  return msg || 'Ошибка при подписании через NCALayer.'
}
