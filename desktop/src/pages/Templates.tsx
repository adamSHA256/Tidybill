import {
  Title,
  Text,
  Paper,
  Stack,
  Group,
  Badge,
  Button,
  Switch,
  Loader,
  Center,
  SimpleGrid,
  TextInput,
  Modal,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { IconEye, IconStar, IconStarFilled, IconCopy, IconCode, IconTrash, IconLock } from '@tabler/icons-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, getApiBase, openInBrowser, type PDFTemplate } from '../api/client'
import { useT } from '../i18n'

export function Templates() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [generatingAll, setGeneratingAll] = useState(false)
  const [generatingId, setGeneratingId] = useState<string | null>(null)
  const [duplicateModal, setDuplicateModal] = useState<string | null>(null)
  const [duplicateName, setDuplicateName] = useState('')
  const { t } = useT()

  const { data: templates, isLoading } = useQuery({
    queryKey: ['templates'],
    queryFn: api.getTemplates,
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<PDFTemplate> }) =>
      api.updateTemplate(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const setDefaultMutation = useMutation({
    mutationFn: api.setDefaultTemplate,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      notifications.show({ title: t('notify.template_default_set'), message: t('notify.template_default_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const duplicateMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      api.duplicateTemplate(id, name),
    onSuccess: (newTmpl) => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      setDuplicateModal(null)
      setDuplicateName('')
      notifications.show({ title: t('template.duplicated'), message: newTmpl.name, color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: api.deleteTemplate,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      notifications.show({ title: t('template.deleted'), message: '', color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const handleGeneratePreview = async (id: string) => {
    setGeneratingId(id)
    try {
      await api.generatePreview(id)
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      notifications.show({ title: t('notify.preview_generated'), message: t('notify.preview_generated_msg'), color: 'green' })
      await openInBrowser(`${getApiBase()}/templates/${id}/preview-pdf`)
    } catch (err) {
      notifications.show({ title: t('common.error'), message: (err as Error).message, color: 'red' })
    } finally {
      setGeneratingId(null)
    }
  }

  const handleGenerateAll = async () => {
    setGeneratingAll(true)
    try {
      const paths = await api.generateAllPreviews()
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      notifications.show({ title: t('notify.all_previews_generated'), message: t('notify.all_previews_msg'), color: 'green' })
      for (const id of Object.keys(paths)) {
        await openInBrowser(`${getApiBase()}/templates/${id}/preview-pdf`)
      }
    } catch (err) {
      notifications.show({ title: t('common.error'), message: (err as Error).message, color: 'red' })
    } finally {
      setGeneratingAll(false)
    }
  }

  const handleDuplicate = (id: string) => {
    if (duplicateName.trim()) {
      duplicateMutation.mutate({ id, name: duplicateName.trim() })
    }
  }

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <div>
          <Title order={2}>{t('template.title')}</Title>
          <Text c="dimmed" size="sm">{t('template.subtitle')}</Text>
        </div>
        <Button
          onClick={handleGenerateAll}
          loading={generatingAll}
          variant="light"
        >
          {t('template.generate_all')}
        </Button>
      </Group>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="lg">
        {templates?.map((tmpl) => (
          <TemplateCard
            key={tmpl.id}
            template={tmpl}
            t={t}
            onUpdate={(data) => updateMutation.mutate({ id: tmpl.id, data })}
            onSetDefault={() => setDefaultMutation.mutate(tmpl.id)}
            onGeneratePreview={() => handleGeneratePreview(tmpl.id)}
            onDuplicate={() => { setDuplicateModal(tmpl.id); setDuplicateName(tmpl.name + ` (${t('template.copy_suffix')})`) }}
            onEditCode={() => navigate(`/template-editor/${tmpl.id}`)}
            onDelete={() => { if (window.confirm(t('template.delete_confirm').replace('{name}', tmpl.name))) deleteMutation.mutate(tmpl.id) }}
            generating={generatingId === tmpl.id}
          />
        ))}
      </SimpleGrid>

      <Modal
        opened={duplicateModal !== null}
        onClose={() => setDuplicateModal(null)}
        title={t('template.duplicate_title')}
      >
        <Stack>
          <TextInput
            label={t('template.new_name')}
            value={duplicateName}
            onChange={(e) => setDuplicateName(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter' && duplicateModal) handleDuplicate(duplicateModal) }}
            autoFocus
          />
          <Group justify="flex-end">
            <Button variant="light" onClick={() => setDuplicateModal(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={() => duplicateModal && handleDuplicate(duplicateModal)}
              loading={duplicateMutation.isPending}
            >
              {t('template.duplicate_btn')}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}

function TemplateCard({
  template: tmpl,
  t,
  onUpdate,
  onSetDefault,
  onGeneratePreview,
  onDuplicate,
  onEditCode,
  onDelete,
  generating,
}: {
  template: PDFTemplate
  t: (key: string) => string
  onUpdate: (data: Partial<PDFTemplate>) => void
  onSetDefault: () => void
  onGeneratePreview: () => void
  onDuplicate: () => void
  onEditCode: () => void
  onDelete: () => void
  generating: boolean
}) {
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(tmpl.name)

  const handleNameSave = () => {
    if (name !== tmpl.name) {
      onUpdate({ name })
    }
    setEditing(false)
  }

  return (
    <Paper p="md" radius="md" withBorder>
      <Stack gap="sm">
        <Group justify="space-between">
          <Group gap="xs">
            {editing ? (
              <TextInput
                size="sm"
                value={name}
                onChange={(e) => setName(e.target.value)}
                onBlur={handleNameSave}
                onKeyDown={(e) => { if (e.key === 'Enter') handleNameSave() }}
                autoFocus
                styles={{ input: { fontWeight: 600 } }}
              />
            ) : tmpl.is_builtin ? (
              <Text fw={600} size="lg">
                {tmpl.name}
              </Text>
            ) : (
              <Text fw={600} size="lg" onClick={() => setEditing(true)} style={{ cursor: 'pointer' }}>
                {tmpl.name}
              </Text>
            )}
            {tmpl.is_builtin && (
              <IconLock size={14} style={{ opacity: 0.4 }} />
            )}
          </Group>
          <Group gap={4}>
            {tmpl.is_default && (
              <Badge color="blue" variant="light">{t('template.default')}</Badge>
            )}
            {!tmpl.is_builtin && (
              <Badge color="grape" variant="light" size="sm">{t('template.custom')}</Badge>
            )}
          </Group>
        </Group>

        <Text c="dimmed" size="sm">{tmpl.description}</Text>
        <Text c="dimmed" size="xs">{t('template.code').replace('{code}', tmpl.template_code)}</Text>

        <Group gap="lg">
          <Switch
            label={t('template.logo')}
            checked={tmpl.show_logo}
            onChange={(e) => onUpdate({ show_logo: e.currentTarget.checked })}
            size="sm"
          />
          <Switch
            label={t('template.qr_code')}
            checked={tmpl.show_qr}
            onChange={(e) => onUpdate({ show_qr: e.currentTarget.checked })}
            size="sm"
          />
          <Switch
            label={t('template.notes')}
            checked={tmpl.show_notes}
            onChange={(e) => onUpdate({ show_notes: e.currentTarget.checked })}
            size="sm"
          />
        </Group>

        <Group gap="xs" mt="xs">
          {!tmpl.is_default && (
            <Button
              size="xs"
              variant="light"
              leftSection={tmpl.is_default ? <IconStarFilled size={14} /> : <IconStar size={14} />}
              onClick={onSetDefault}
            >
              {t('template.set_default')}
            </Button>
          )}
          <Button
            size="xs"
            variant="light"
            color="teal"
            leftSection={<IconEye size={14} />}
            onClick={onGeneratePreview}
            loading={generating}
          >
            {t('template.generate_preview')}
          </Button>
          <Button
            size="xs"
            variant="light"
            color="violet"
            leftSection={<IconCopy size={14} />}
            onClick={onDuplicate}
          >
            {t('template.duplicate_btn')}
          </Button>
          {!tmpl.is_builtin && (
            <>
              <Button
                size="xs"
                variant="light"
                color="indigo"
                leftSection={<IconCode size={14} />}
                onClick={onEditCode}
              >
                {t('template.edit_code')}
              </Button>
              <Button
                size="xs"
                variant="light"
                color="red"
                leftSection={<IconTrash size={14} />}
                onClick={onDelete}
              >
                {t('common.delete')}
              </Button>
            </>
          )}
        </Group>
      </Stack>
    </Paper>
  )
}
