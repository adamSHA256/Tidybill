import { useState, useEffect, useRef } from 'react'
import {
  Stack,
  Text,
  Group,
  Button,
  Badge,
  Modal,
  Alert,
  Checkbox,
  SimpleGrid,
  Paper,
  TextInput,
  ActionIcon,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconShieldLock, IconAlertTriangle, IconCheck, IconEye, IconEyeOff } from '@tabler/icons-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { useT } from '../i18n'

export function MasterKeyPanel() {
  const { t } = useT()
  const qc = useQueryClient()

  const { data: status, isLoading } = useQuery({
    queryKey: ['master-key-status'],
    queryFn: api.masterKey.status,
    refetchInterval: false,
  })

  const configured = status?.configured ?? false

  // ── Generate flow ─────────────────────────────────────────────────────────
  const [generateModalOpen, setGenerateModalOpen] = useState(false)
  const [generatedPhrase, setGeneratedPhrase] = useState<string | null>(null)
  const [phraseWrittenDown, setPhraseWrittenDown] = useState(false)
  const [generating, setGenerating] = useState(false)

  const openGenerate = () => {
    setGeneratedPhrase(null)
    setPhraseWrittenDown(false)
    setGenerateModalOpen(true)
  }

  const handleGenerate = async () => {
    setGenerating(true)
    try {
      const res = await api.masterKey.generate()
      setGeneratedPhrase(res.phrase)
      qc.invalidateQueries({ queryKey: ['master-key-status'] })
    } catch {
      notifications.show({ color: 'red', message: t('master_key.error_keychain') })
    } finally {
      setGenerating(false)
    }
  }

  const closeGenerateModal = () => {
    setGenerateModalOpen(false)
    setGeneratedPhrase(null)
    setPhraseWrittenDown(false)
  }

  // ── Reveal flow ───────────────────────────────────────────────────────────
  const [revealConfirmOpen, setRevealConfirmOpen] = useState(false)
  const [revealedPhrase, setRevealedPhrase] = useState<string | null>(null)
  const [revealCountdown, setRevealCountdown] = useState(30)
  const [revealing, setRevealing] = useState(false)
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const handleReveal = async () => {
    setRevealing(true)
    try {
      const tokenRes = await api.masterKey.revealToken()
      const phraseRes = await api.masterKey.reveal(tokenRes.token)
      setRevealedPhrase(phraseRes.phrase)
      setRevealCountdown(30)
      setRevealConfirmOpen(false)

      countdownRef.current = setInterval(() => {
        setRevealCountdown((c) => {
          if (c <= 1) {
            clearInterval(countdownRef.current!)
            countdownRef.current = null
            setRevealedPhrase(null)
            return 30
          }
          return c - 1
        })
      }, 1000)
    } catch {
      notifications.show({ color: 'red', message: t('master_key.error_keychain') })
    } finally {
      setRevealing(false)
    }
  }

  const hideReveal = () => {
    if (countdownRef.current) clearInterval(countdownRef.current)
    countdownRef.current = null
    setRevealedPhrase(null)
    setRevealCountdown(30)
  }

  useEffect(() => {
    return () => {
      if (countdownRef.current) clearInterval(countdownRef.current)
    }
  }, [])

  // ── Import flow ───────────────────────────────────────────────────────────
  const [importModalOpen, setImportModalOpen] = useState(false)
  const [importWords, setImportWords] = useState<string[]>(Array(12).fill(''))
  const [importing, setImporting] = useState(false)

  const openImport = () => {
    setImportWords(Array(12).fill(''))
    setImportModalOpen(true)
  }

  const handleImportWordChange = (idx: number, value: string) => {
    // Paste detection: if value contains spaces and is pasted into word 0,
    // split across all 12 inputs.
    if (idx === 0 && value.trim().includes(' ')) {
      const words = value.trim().split(/\s+/)
      const filled = [...Array(12).fill('')].map((_, i) => words[i] || '')
      setImportWords(filled)
      return
    }
    setImportWords((prev) => prev.map((w, i) => (i === idx ? value.trim() : w)))
  }

  const handleImportSubmit = async () => {
    const phrase = importWords.map((w) => w.trim()).join(' ')
    setImporting(true)
    try {
      await api.masterKey.import(phrase)
      qc.invalidateQueries({ queryKey: ['master-key-status'] })
      setImportModalOpen(false)
      notifications.show({ color: 'green', message: t('master_key.configured_title') })
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      if (msg.includes('invalid BIP-39')) {
        notifications.show({ color: 'red', message: t('master_key.error_invalid_phrase') })
      } else {
        notifications.show({ color: 'red', message: t('master_key.error_keychain') })
      }
    } finally {
      setImporting(false)
    }
  }

  // ── Delete flow ───────────────────────────────────────────────────────────
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const deleteMutation = useMutation({
    mutationFn: api.masterKey.delete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['master-key-status'] })
      setDeleteModalOpen(false)
    },
    onError: () => {
      notifications.show({ color: 'red', message: t('master_key.error_keychain') })
    },
  })

  if (isLoading) return null

  const words = generatedPhrase?.split(' ') ?? []
  const revealWords = revealedPhrase?.split(' ') ?? []

  return (
    <Stack gap="md">
      <Group justify="space-between" align="center">
        <Group gap="xs">
          <IconShieldLock size={18} />
          <Text fw={500}>{t('master_key.section_title')}</Text>
        </Group>
        {configured ? (
          <Badge color="green" size="sm">{t('master_key.configured_title')}</Badge>
        ) : (
          <Badge color="yellow" size="sm">{t('master_key.not_configured_title')}</Badge>
        )}
      </Group>

      {!configured ? (
        <>
          <Text size="sm" c="dimmed">{t('master_key.not_configured_desc')}</Text>
          <Group>
            <Button size="sm" onClick={openGenerate}>{t('master_key.generate_btn')}</Button>
            <Button size="sm" variant="default" onClick={openImport}>{t('master_key.import_btn')}</Button>
          </Group>
        </>
      ) : (
        <>
          <Text size="sm" c="dimmed">{t('master_key.configured_desc')}</Text>

          {/* Revealed phrase display */}
          {revealedPhrase && (
            <Paper withBorder p="md" radius="md">
              <Group justify="space-between" mb="sm">
                <Text size="sm" fw={500}>{t('master_key.phrase_grid_title')}</Text>
                <Group gap="xs">
                  <Text size="xs" c="dimmed">
                    {t('master_key.reveal_autohide').replace('{seconds}', String(revealCountdown))}
                  </Text>
                  <ActionIcon variant="subtle" size="sm" onClick={hideReveal}>
                    <IconEyeOff size={14} />
                  </ActionIcon>
                </Group>
              </Group>
              <SimpleGrid cols={3} spacing="xs">
                {revealWords.map((word, i) => (
                  <Paper key={i} withBorder p="xs" radius="sm" style={{ textAlign: 'center' }}>
                    <Text size="xs" c="dimmed" mb={2}>{i + 1}</Text>
                    <Text ff="monospace" size="sm" fw={600}>{word}</Text>
                  </Paper>
                ))}
              </SimpleGrid>
            </Paper>
          )}

          <Group>
            {!revealedPhrase && (
              <Button
                size="sm"
                variant="default"
                leftSection={<IconEye size={14} />}
                onClick={() => setRevealConfirmOpen(true)}
              >
                {t('master_key.reveal_btn')}
              </Button>
            )}
            <Button size="sm" variant="default" onClick={openImport}>{t('master_key.change_btn')}</Button>
            <Button size="sm" color="red" variant="subtle" onClick={() => setDeleteModalOpen(true)}>
              {t('master_key.delete_btn')}
            </Button>
          </Group>
        </>
      )}

      {/* ── Generate modal ──────────────────────────────────────────────── */}
      <Modal
        opened={generateModalOpen}
        onClose={closeGenerateModal}
        title={t('master_key.generate_modal_title')}
        centered
        size={generatedPhrase ? 'lg' : 'md'}
      >
        {!generatedPhrase ? (
          <Stack>
            <Alert color="yellow" icon={<IconAlertTriangle size={16} />}>
              {t('master_key.generate_modal_warn')}
            </Alert>
            <Group justify="flex-end">
              <Button variant="default" onClick={closeGenerateModal}>{t('action.cancel')}</Button>
              <Button loading={generating} onClick={handleGenerate}>
                {t('master_key.generate_confirm')}
              </Button>
            </Group>
          </Stack>
        ) : (
          <Stack>
            <Text fw={500}>{t('master_key.phrase_grid_title')}</Text>
            <Alert color="orange" icon={<IconAlertTriangle size={16} />}>
              {t('master_key.phrase_grid_warn')}
            </Alert>
            <SimpleGrid cols={3} spacing="xs">
              {words.map((word, i) => (
                <Paper key={i} withBorder p="xs" radius="sm" style={{ textAlign: 'center' }}>
                  <Text size="xs" c="dimmed" mb={2}>{i + 1}</Text>
                  <Text ff="monospace" size="sm" fw={600}>{word}</Text>
                </Paper>
              ))}
            </SimpleGrid>
            <Checkbox
              label={t('master_key.phrase_written_checkbox')}
              checked={phraseWrittenDown}
              onChange={(e) => setPhraseWrittenDown(e.currentTarget.checked)}
            />
            <Group justify="flex-end">
              <Button
                disabled={!phraseWrittenDown}
                leftSection={<IconCheck size={14} />}
                onClick={closeGenerateModal}
              >
                {t('master_key.done_btn')}
              </Button>
            </Group>
          </Stack>
        )}
      </Modal>

      {/* ── Reveal confirm modal ─────────────────────────────────────────── */}
      <Modal
        opened={revealConfirmOpen}
        onClose={() => setRevealConfirmOpen(false)}
        title={t('master_key.reveal_modal_title')}
        centered
      >
        <Stack>
          <Alert color="orange" icon={<IconAlertTriangle size={16} />}>
            {t('master_key.reveal_modal_warn')}
          </Alert>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setRevealConfirmOpen(false)}>{t('action.cancel')}</Button>
            <Button loading={revealing} leftSection={<IconEye size={14} />} onClick={handleReveal}>
              {t('master_key.reveal_confirm')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* ── Import modal ─────────────────────────────────────────────────── */}
      <Modal
        opened={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        title={t('master_key.import_modal_title')}
        centered
        size="lg"
      >
        <Stack>
          <Text size="sm" c="dimmed">{t('master_key.import_modal_desc')}</Text>
          <SimpleGrid cols={3} spacing="xs">
            {importWords.map((word, i) => (
              <TextInput
                key={i}
                label={t('master_key.import_word_placeholder').replace('{n}', String(i + 1))}
                value={word}
                onChange={(e) => handleImportWordChange(i, e.currentTarget.value)}
                ff="monospace"
                size="sm"
              />
            ))}
          </SimpleGrid>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setImportModalOpen(false)}>{t('action.cancel')}</Button>
            <Button
              loading={importing}
              disabled={importWords.some((w) => !w.trim())}
              onClick={handleImportSubmit}
            >
              {t('master_key.import_confirm')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* ── Delete modal ─────────────────────────────────────────────────── */}
      <Modal
        opened={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title={t('master_key.delete_modal_title')}
        centered
      >
        <Stack>
          <Alert color="red" icon={<IconAlertTriangle size={16} />}>
            {t('master_key.delete_modal_warn')}
          </Alert>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setDeleteModalOpen(false)}>{t('action.cancel')}</Button>
            <Button
              color="red"
              loading={deleteMutation.isPending}
              onClick={() => deleteMutation.mutate()}
            >
              {t('master_key.delete_confirm')}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
