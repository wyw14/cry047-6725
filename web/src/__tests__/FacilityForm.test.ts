import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import FacilityForm from '../components/FacilityForm.vue';
import type { Place, ResponsiblePerson } from '../types/models';

const places: Place[] = [
  { id: 'p1', name: '图书馆', type: 'library', address: '市中心', code: 'LIB-001', created_at: '', updated_at: '', version: 1 },
];
const people: ResponsiblePerson[] = [
  { id: 'rp1', name: '张伟', email: 'z@x.com', phone: '13800138000', department: '运维一部', active: true, created_at: '', updated_at: '', version: 1 },
];

describe('FacilityForm', () => {
  it('emits submit with the form payload', async () => {
    const wrapper = mount(FacilityForm, { props: { places, people } });
    const form = wrapper.find('form');
    await form.trigger('submit.prevent');
    const submitEvents = wrapper.emitted('submit');
    expect(submitEvents).toBeTruthy();
    expect(submitEvents![0][0]).toMatchObject({
      place_id: '',
      name: '',
      code: '',
      criticality: 'standard',
      description: '',
    });
  });

  it('emits cancel when the cancel button is clicked', async () => {
    const wrapper = mount(FacilityForm, { props: { places, people } });
    const buttons = wrapper.findAll('button');
    const cancelBtn = buttons.find((b) => b.text().includes('取消'));
    expect(cancelBtn).toBeTruthy();
    await cancelBtn!.trigger('click');
    expect(wrapper.emitted('cancel')).toBeTruthy();
  });
});
