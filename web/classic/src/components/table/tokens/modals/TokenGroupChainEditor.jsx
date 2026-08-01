/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import React, { useMemo } from 'react';
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Select, Tag, Typography } from '@douyinfe/semi-ui';
import { GripVertical, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { renderGroupOption } from '../../../../helpers';

const { Text } = Typography;

function SortableGroup({ group, index, option, canRemove, onRemove }) {
  const { t } = useTranslation();
  const sortable = useSortable({ id: group });
  const style = {
    transform: CSS.Transform.toString(sortable.transform),
    transition: sortable.transition,
    zIndex: sortable.isDragging ? 10 : undefined,
    boxShadow: sortable.isDragging ? '0 8px 24px rgb(0 0 0 / 12%)' : undefined,
  };
  const forwardSortableArrowKey = (event) => {
    if (
      !sortable.isDragging ||
      !['ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowUp'].includes(event.code)
    ) {
      return;
    }

    event.preventDefault();
    document.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: event.key,
        code: event.code,
        bubbles: true,
        cancelable: true,
      }),
    );
  };

  return (
    <div
      ref={sortable.setNodeRef}
      style={style}
      className='flex min-h-14 w-full min-w-0 items-center gap-3 bg-white px-3 py-2'
    >
      <button
        ref={sortable.setActivatorNodeRef}
        type='button'
        className='flex h-8 w-8 shrink-0 touch-none cursor-grab items-center justify-center rounded-md border-0 bg-transparent p-0 text-gray-500 hover:bg-gray-100 active:cursor-grabbing'
        aria-label={t('拖动调整优先级')}
        title={t('拖动调整优先级')}
        onKeyDownCapture={forwardSortableArrowKey}
        {...sortable.attributes}
        {...sortable.listeners}
      >
        <GripVertical size={16} />
      </button>
      <Tag size='small' color='blue'>
        {index + 1}
      </Tag>
      <span className='min-w-0 flex-1'>
        <Text strong className='block truncate'>
          {group}
        </Text>
        {(option?.desc ||
          (option?.label !== group ? option?.label : undefined)) && (
          <Text type='tertiary' size='small' className='block truncate'>
            {option.desc || option.label}
          </Text>
        )}
      </span>
      {option?.ratio !== undefined && (
        <Tag size='small' color='green'>
          {option.ratio}x
        </Tag>
      )}
      <button
        type='button'
        className='flex h-8 w-8 shrink-0 items-center justify-center rounded-md border-0 bg-transparent p-0 text-red-500 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50'
        disabled={!canRemove}
        aria-label={t('移除分组')}
        title={t('移除分组')}
        onClick={onRemove}
      >
        <Trash2 size={16} />
      </button>
    </div>
  );
}

export default function TokenGroupChainEditor({
  options,
  value = [],
  onChange,
}) {
  const { t } = useTranslation();
  const optionMap = useMemo(
    () => new Map(options.map((option) => [option.value, option])),
    [options],
  );
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const handleSelectionChange = (nextValue) => {
    if (!Array.isArray(nextValue) || nextValue.length === 0) return;
    const selected = new Set(nextValue);
    const retained = value.filter((group) => selected.has(group));
    const added = nextValue.filter((group) => !value.includes(group));
    onChange([...retained, ...added]);
  };

  const handleDragEnd = (event) => {
    if (!event.over || event.active.id === event.over.id) return;
    const oldIndex = value.indexOf(String(event.active.id));
    const newIndex = value.indexOf(String(event.over.id));
    if (oldIndex < 0 || newIndex < 0) return;
    onChange(arrayMove(value, oldIndex, newIndex));
  };

  return (
    <div className='min-w-0 space-y-3'>
      <Select
        multiple
        filter
        value={value}
        optionList={options}
        renderOptionItem={renderGroupOption}
        placeholder={t('选择分组')}
        onChange={handleSelectionChange}
        searchPosition='dropdown'
        autoClearSearchValue={false}
        showClear={false}
        style={{ width: '100%' }}
        disabled={options.length === 0}
      />

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext items={value} strategy={verticalListSortingStrategy}>
          <div className='w-full min-w-0 divide-y divide-gray-100 overflow-hidden rounded-lg border border-solid border-gray-200'>
            {value.map((group, index) => (
              <SortableGroup
                key={group}
                group={group}
                index={index}
                option={optionMap.get(group)}
                canRemove={value.length > 1}
                onRemove={() =>
                  onChange(value.filter((item) => item !== group))
                }
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>
    </div>
  );
}
