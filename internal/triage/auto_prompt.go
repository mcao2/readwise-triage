package triage

// AutoTriagePromptTemplate is a lean prompt for automated LLM triage.
// It only requests fields that are actually consumed downstream (action,
// reason, suggested_tags), saving tokens compared to the full export prompt.
const AutoTriagePromptTemplate = `You are my personal reading assistant. I will give you a batch of Readwise Reader inbox item metadata (JSON format). Classify each item with a triage decision.

---

**My Reading Goals:**
- Primary: Tool usage guides, productivity tips, actionable methodologies
- Secondary: Industry insights, technical deep-dives
- Usually ignore: Pure opinion pieces, marketing content, outdated info

---

**Output the following structure for each item (JSON format):**

{
  "id": "item id",
  "title": "title",
  "url": "url",
  
  "triage_decision": {
    "action": "delete|archive|later|read_now",
    "reason": "why this classification (1-2 sentences)"
  },
  
  "metadata_enhancement": {
    "suggested_tags": ["tag1", "tag2"]
  }
}

---

**Special Rules:**
1. **action = "read_now"**: Only for items that are highly actionable, from credible sources, and can solve problems I might currently face.
2. **action = "later"**: Valuable but not urgent, or requires a full time block.
3. **action = "archive"**: Might be useful later but don't need deep reading now.
4. **action = "delete"**: Marketing content, duplicates, outdated info, clearly irrelevant.

---

**Output Format:**
Return ONLY a JSON array, each element is the above format. No additional text, commentary, or summaries outside the JSON.

---

**Inbox items to process:**

%s`
